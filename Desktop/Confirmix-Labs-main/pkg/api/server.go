package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
	"github.com/google/uuid"
	"confirmix/pkg/types"
)

// WebServer represents the web server instance
type WebServer struct {
	blockchain      *blockchain.Blockchain
	consensusEngine *consensus.HybridConsensus
	validatorManager *consensus.ValidatorManager
	governance      *consensus.Governance
	port           int
	router         *mux.Router
	server         *http.Server  // Add server field
	
	// Önbellek verileri
	validatorsCache      []blockchain.ValidatorInfo
	validatorsCacheTime  time.Time
	validatorsCacheMutex sync.RWMutex
	
	// İşlemler için önbellek
	transactionsCache      []*blockchain.Transaction
	transactionsCacheTime  time.Time
	transactionsCacheMutex sync.RWMutex
	
	// Bekleyen işlemler için ayrı önbellek
	pendingTxCache      []*blockchain.Transaction
	pendingTxCacheTime  time.Time
	pendingTxCacheMutex sync.RWMutex
	
	// Onaylanmış işlemler için ayrı önbellek
	confirmedTxCache      []*blockchain.Transaction
	confirmedTxCacheTime  time.Time
	confirmedTxCacheMutex sync.RWMutex
	
	// Genel blockchain önbellekleri
	
	// Blok önbelleği - key: block index, value: *blockchain.Block
	blockCache       sync.Map // thread-safe map
	blockCacheExpiry sync.Map // ne zaman süresi dolacak
	
	// Bakiye önbelleği - key: address, value: *big.Int
	balanceCache       sync.Map
	balanceCacheExpiry sync.Map
}

// NewWebServer creates a new web server instance
func NewWebServer(bc *blockchain.Blockchain, ce *consensus.HybridConsensus, vm *consensus.ValidatorManager, gov *consensus.Governance, port int) *WebServer {
	ws := &WebServer{
		blockchain:      bc,
		consensusEngine: ce,
		validatorManager: vm,
		governance:      gov,
		port:           port,
		router:         mux.NewRouter(),
	}
	ws.setupRoutes()
	return ws
}

// enableCORS enables CORS for all routes
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures the HTTP routes
func (ws *WebServer) setupRoutes() {
	ws.router = mux.NewRouter()
	
	// Enable CORS for all routes
	ws.router.Use(enableCORS)

	// Blockchain routes
	ws.router.HandleFunc("/api/status", ws.getStatus).Methods("GET")
	ws.router.HandleFunc("/api/blocks", ws.getBlocks).Methods("GET")
	ws.router.HandleFunc("/api/blocks/{index}", ws.getBlockByIndex).Methods("GET")
	ws.router.HandleFunc("/api/transactions", ws.getAllTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions/pending", ws.getPendingTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions/confirmed", ws.getConfirmedTransactions).Methods("GET")
	ws.router.HandleFunc("/api/transactions", ws.createTransaction).Methods("POST")
	ws.router.HandleFunc("/api/blockchain/transactions/{hash}/revert", ws.revertTransaction).Methods("POST")
	
	// Wallet routes
	ws.router.HandleFunc("/api/wallet/create", ws.createWallet).Methods("POST")
	ws.router.HandleFunc("/api/wallet/import", ws.importWallet).Methods("POST")
	ws.router.HandleFunc("/api/wallet/balance/{address}", ws.getWalletBalance).Methods("GET")
	ws.router.HandleFunc("/api/wallet/balance/{address}/simple", ws.getWalletBalanceSimple).Methods("GET")
	ws.router.HandleFunc("/api/wallet/transfer", ws.transfer).Methods("POST")
	
	// Mining routes
	ws.router.HandleFunc("/api/mine", ws.mineBlock).Methods("POST")
	
	// Validator routes
	ws.router.HandleFunc("/api/validators", ws.getValidators).Methods("GET")
	ws.router.HandleFunc("/api/validators/register", ws.registerValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/approve", ws.approveValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/reject", ws.rejectValidator).Methods("POST")
	ws.router.HandleFunc("/api/validators/suspend", ws.suspendValidator).Methods("POST")
	
	// Admin routes
	ws.router.HandleFunc("/api/admin/add", ws.addAdmin).Methods("POST")
	ws.router.HandleFunc("/api/admin/remove", ws.removeAdmin).Methods("POST")
	ws.router.HandleFunc("/api/admin/list", ws.listAdmins).Methods("GET")
	
	// Governance routes
	ws.router.HandleFunc("/api/proposals", ws.listProposals).Methods("GET")
	ws.router.HandleFunc("/api/proposals/{id}", ws.getProposal).Methods("GET")
	ws.router.HandleFunc("/api/proposals/create", ws.createProposal).Methods("POST")
	ws.router.HandleFunc("/api/proposals/vote", ws.castVote).Methods("POST")
	
	// Health check
	ws.router.HandleFunc("/api/health", ws.getHealthCheck).Methods("GET")

	// Multi-signature routes
	ws.router.HandleFunc("/api/multisig/wallet/create", ws.createMultiSigWallet).Methods("POST")
	ws.router.HandleFunc("/api/multisig/wallet/{address}", ws.getMultiSigWallet).Methods("GET")
	ws.router.HandleFunc("/api/multisig/transaction/create", ws.createMultiSigTransaction).Methods("POST")
	ws.router.HandleFunc("/api/multisig/transaction/sign", ws.signMultiSigTransaction).Methods("POST")
	ws.router.HandleFunc("/api/multisig/transaction/execute", ws.executeMultiSigTransaction).Methods("POST")
	ws.router.HandleFunc("/api/multisig/transaction/{walletAddress}/{txID}/status", ws.getMultiSigTransactionStatus).Methods("GET")
	ws.router.HandleFunc("/api/multisig/transaction/{walletAddress}/pending", ws.getMultiSigPendingTransactions).Methods("GET")
}

// Start starts the web server
func (ws *WebServer) Start() error {
	// Preload cache data
	log.Printf("Preloading caches for better performance...")
	ws.PreloadCache()
	
	// Start the server
	addr := fmt.Sprintf(":%d", ws.port)
	log.Printf("Web server listening on %s", addr)
	
	// Create and store the server instance
	ws.server = &http.Server{
		Addr:    addr,
		Handler: ws.router,
	}
	
	return ws.server.ListenAndServe()
}

// PreloadCache pre-populates cache to avoid initial timeouts
func (ws *WebServer) PreloadCache() {
	log.Printf("Starting cache preloading (simplified)...")
	startTime := time.Now()
	
	// Önce blokzincirdeki önemli adresleri alarak bakiye önbelleğini dolduralım
	addresses := ws.blockchain.GetAllAddresses()
	if len(addresses) > 0 {
		log.Printf("Found %d addresses in blockchain (including genesis and node address)", len(addresses))
		
		// Limit the number of addresses to preload
		preloadCount := 10
		if len(addresses) < preloadCount {
			preloadCount = len(addresses)
		}
		
		preloadAddresses := addresses[:preloadCount]
		
		// Create static cache data
		for _, addr := range preloadAddresses {
			// Default balance 0 tokens
			ws.balanceCache.Store(addr, big.NewInt(0))
			ws.balanceCacheExpiry.Store(addr, time.Now().Add(60*time.Second))
		}
		
		log.Printf("Preloaded %d address balances with default value in %v", 
			preloadCount, time.Since(startTime))
	} else {
		log.Printf("No addresses found to preload balances for")
	}
	
	// Validators - Set a static default list first for immediate use
	defaultValidators := []blockchain.ValidatorInfo{}
	
	// First fill the cache with default values
	ws.validatorsCacheMutex.Lock()
	ws.validatorsCache = defaultValidators 
	ws.validatorsCacheTime = time.Now()
	ws.validatorsCacheMutex.Unlock()
	
	// Try to get real validators in the background
	go func() {
		validatorStart := time.Now()
		log.Printf("Background fetching registered validators...")
		
		// Try with a short timeout - but in background to not delay page load
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		// Use a different channel for communication
		done := make(chan bool, 1)
		var validators []blockchain.ValidatorInfo
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in validator preloading: %v", r)
				}
				done <- true
			}()
			
			// Get validators from blockchain
			validators = ws.blockchain.GetValidators()
			
			if len(validators) > 0 {
				// Update cache
				ws.validatorsCacheMutex.Lock()
				ws.validatorsCache = validators
				ws.validatorsCacheTime = time.Now()
				ws.validatorsCacheMutex.Unlock()
				
				log.Printf("Background loaded %d registered validators in %v", 
					len(validators), time.Since(validatorStart))
			} else {
				log.Printf("No registered validators found in blockchain")
			}
		}()
		
		// Wait for either completion or timeout
		select {
		case <-done:
			// İşlem tamamlandı
		case <-ctx.Done():
			log.Printf("Timeout preloading validators: %v", ctx.Err())
		}
	}()
	
	// Boş transaction listeleri oluşturalım - sonra arkaplanda gerçekleri almayı deneriz
	emptyTxs := make([]*blockchain.Transaction, 0)
	
	// Boş bekleyen işlem listesi
	ws.pendingTxCacheMutex.Lock()
	ws.pendingTxCache = emptyTxs
	ws.pendingTxCacheTime = time.Now()
	ws.pendingTxCacheMutex.Unlock()
	
	// Boş onaylanmış işlem listesi
	ws.confirmedTxCacheMutex.Lock()
	ws.confirmedTxCache = emptyTxs  
	ws.confirmedTxCacheTime = time.Now()
	ws.confirmedTxCacheMutex.Unlock()
	
	// Boş tüm işlemler listesi
	ws.transactionsCacheMutex.Lock()
	ws.transactionsCache = emptyTxs
	ws.transactionsCacheTime = time.Now()
	ws.transactionsCacheMutex.Unlock()
	
	log.Printf("Initialized empty transaction lists")
	
	// Arkaplanda işlemleri getirmeye çalışalım
	go func() {
		txStart := time.Now()
		
		// Bekleyen işlemleri al
		pending := ws.blockchain.GetPendingTransactions()
		pendingWithStatus := make([]*blockchain.Transaction, 0, len(pending))
		
		for _, tx := range pending {
			txCopy := *tx
			txCopy.Status = "pending"
			pendingWithStatus = append(pendingWithStatus, &txCopy)
		}
		
		// Sadece boş değilse güncelle
		if len(pendingWithStatus) > 0 {
			ws.pendingTxCacheMutex.Lock()
			ws.pendingTxCache = pendingWithStatus
			ws.pendingTxCacheTime = time.Now()
			ws.pendingTxCacheMutex.Unlock()
			
			log.Printf("Background loaded %d pending transactions in %v", 
				len(pendingWithStatus), time.Since(txStart))
				
			// Combined transactions listesini de güncelle
			ws.transactionsCacheMutex.Lock()
			ws.transactionsCache = pendingWithStatus // Başlangıç için sadece bekleyenler
			ws.transactionsCacheTime = time.Now()
			ws.transactionsCacheMutex.Unlock()
		}
		
		// İşimiz bitti, onaylanmış işlemleri daha sonra lazım olursa getireceğiz
		log.Printf("Transaction preloading completed in %v", time.Since(txStart))
	}()
	
	log.Printf("Cache preloading initiated in %v", time.Since(startTime))
}

// getStatus handles the status endpoint
func (ws *WebServer) getStatus(w http.ResponseWriter, r *http.Request) {
	// Set headers for CORS
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Create a static status response
	// Using cached or default values to avoid blockchain calls
	status := struct {
		Status   string `json:"status"`
		Height   uint64 `json:"height"`
		Uptime   string `json:"uptime"`
		Version  string `json:"version"`
		NodeType string `json:"nodeType"`
	}{
		Status:   "online",
		Height:   ws.blockchain.GetChainHeight(),
		Uptime:   "active",
		Version:  "1.0.0",
		NodeType: "validator",
	}
	
	// Always return OK
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// getBlocks handles the blocks endpoint
func (ws *WebServer) getBlocks(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse limit from query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Cap limit at 50
	if limit > 50 {
		limit = 50
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	blocksChan := make(chan []struct {
		Index        uint64 `json:"Index"`
		Timestamp    int64  `json:"Timestamp"`
		Hash         string `json:"Hash"`
		PrevHash     string `json:"PrevHash"`
		Validator    string `json:"Validator"`
		Transactions int    `json:"Transactions"`
	}, 1)
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getBlocks: %v", r)
			}
			done <- true
		}()
		
		log.Printf("Getting blocks from blockchain, limit=%d", limit)
		
		// Get chain height safely as int (not uint64)
		chainHeight := int(ws.blockchain.GetChainHeight())
		
		// Create result array
		blocksResponse := make([]struct {
			Index        uint64 `json:"Index"`
			Timestamp    int64  `json:"Timestamp"`
			Hash         string `json:"Hash"`
			PrevHash     string `json:"PrevHash"`
			Validator    string `json:"Validator"`
			Transactions int    `json:"Transactions"`
		}, 0, limit)
		
		// Start from the most recent block and go backwards
		// Ensure we don't go negative or exceed the chain height
		for i := chainHeight; i >= 0 && len(blocksResponse) < limit; i-- {
			// Convert index to uint64 only when passing to blockchain API
			blockIndex := uint64(i)
			blockIndexKey := fmt.Sprintf("block_%d", blockIndex)
			
			var block *blockchain.Block
			var err error
			
			// First check cache
			if cachedValue, ok := ws.blockCache.Load(blockIndexKey); ok {
				if expiryTime, ok := ws.blockCacheExpiry.Load(blockIndexKey); ok {
					if time.Now().Before(expiryTime.(time.Time)) {
						// Cached value is still valid
						block = cachedValue.(*blockchain.Block)
					}
				}
			}
			
			// If not in cache, get from blockchain
			if block == nil {
				block, err = ws.blockchain.GetBlockByIndex(blockIndex)
				if err != nil {
					log.Printf("Error getting block at index %d: %v", i, err)
					continue
				}
				
				// Cache block for future use (blocks don't change)
				ws.blockCache.Store(blockIndexKey, block)
				ws.blockCacheExpiry.Store(blockIndexKey, time.Now().Add(60*time.Second))
			}
			
			// Make sure block has valid Hash field
			if block != nil {
				blockHash := block.Hash
				if blockHash == "" {
					// Generate a hash if missing
					blockHash = fmt.Sprintf("block_%d_%d", block.Index, block.Timestamp)
				}
				
				blockResp := struct {
					Index        uint64 `json:"Index"`
					Timestamp    int64  `json:"Timestamp"`
					Hash         string `json:"Hash"`
					PrevHash     string `json:"PrevHash"`
					Validator    string `json:"Validator"`
					Transactions int    `json:"Transactions"`
				}{
					Index:        block.Index,
					Timestamp:    block.Timestamp,
					Hash:         blockHash,
					PrevHash:     block.PrevHash,
					Validator:    block.Validator,
					Transactions: len(block.Transactions),
				}
				
				blocksResponse = append(blocksResponse, blockResp)
			}
		}
		
		log.Printf("Retrieved %d blocks", len(blocksResponse))
		blocksChan <- blocksResponse
	}()
	
	// Wait for completion or timeout
	select {
	case blocks := <-blocksChan:
		w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(blocks)
		
	case <-done:
		// No blocks sent, return empty array
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		
	case <-ctx.Done():
		// Timeout - return what we have
		log.Printf("Timeout getting blocks: %v", ctx.Err())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}
}

// getPendingTransactions handles the pending transactions endpoint with caching
func (ws *WebServer) getPendingTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers and Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get limit parameter, default to 50
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// Cap the limit to prevent performance issues
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}
	
	// Önbellekteki verileri kontrol edelim (5 saniyeden daha yeni ise)
	ws.pendingTxCacheMutex.RLock()
	cacheAge := time.Since(ws.pendingTxCacheTime)
	hasCache := len(ws.pendingTxCache) > 0 && cacheAge < 5*time.Second
	
	// Eğer önbellekte güncel veri varsa ve istenen limit önbellek boyutundan az veya eşitse, hemen döndürelim
	if hasCache && limit <= len(ws.pendingTxCache) {
		// Önbellekten limiti kadar veri alalım
		txs := ws.pendingTxCache
		if limit < len(txs) {
			txs = txs[:limit]
		}
		ws.pendingTxCacheMutex.RUnlock()
		
		log.Printf("Returning %d pending transactions from cache (age: %v)", len(txs), cacheAge)
		w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txs)
		return
	}
	ws.pendingTxCacheMutex.RUnlock()
	
	// Önbellekte veri yoksa veya eski ise veya istenen limit önbellek boyutundan büyükse, yeni veri alalım
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second) // Timeout süresini 3 saniyeden 10 saniyeye çıkardık
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var pendingTxs []*blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getPendingTransactions: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		startTime := time.Now()
		log.Printf("Getting pending transactions from blockchain")
		
		// Get all pending transactions
		pendingTxsRaw := ws.blockchain.GetPendingTransactions()
		
		// Create a new array to hold our response and make a deep copy to avoid race conditions
		pendingTxs = make([]*blockchain.Transaction, 0, len(pendingTxsRaw))
		
		for _, tx := range pendingTxsRaw {
			// Create a copy of each transaction
			txCopy := *tx
			// Add a status field for pending transactions
			txCopy.Status = "pending"
			pendingTxs = append(pendingTxs, &txCopy)
		}
		
		// Önbelleği güncelle
		ws.pendingTxCacheMutex.Lock()
		ws.pendingTxCache = pendingTxs
		ws.pendingTxCacheTime = time.Now()
		ws.pendingTxCacheMutex.Unlock()
		
		log.Printf("Retrieved %d pending transactions in %v", len(pendingTxs), time.Since(startTime))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting pending transactions: %v", err)
			// Return an empty array instead of an error
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
			return
		}
		
		// Create empty array if nil
		if pendingTxs == nil {
			pendingTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Limit the response if needed
		if len(pendingTxs) > limit {
			pendingTxs = pendingTxs[:limit]
		}
		
		// Return the transactions as JSON
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(pendingTxs)
		if err != nil {
			log.Printf("Error encoding pending transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		return
		
	case <-ctx.Done():
		log.Printf("Timeout getting pending transactions: %v", ctx.Err())
		
		// Önbellekte herhangi bir veri varsa, eski de olsa döndürelim
		ws.pendingTxCacheMutex.RLock()
		hasCacheData := len(ws.pendingTxCache) > 0
		cachedTxs := ws.pendingTxCache // Kopyasını alalım
		if limit < len(cachedTxs) {
			cachedTxs = cachedTxs[:limit]
		}
		ws.pendingTxCacheMutex.RUnlock()
		
		if hasCacheData {
			log.Printf("Returning %d pending transactions from stale cache due to timeout", len(cachedTxs))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cachedTxs)
			return
		}
		
		// Hiç önbellek verisi yoksa boş dizi döndür
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
	}
}

// createTransaction handles the transaction creation endpoint
func (ws *WebServer) createTransaction(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var simpleTransaction *blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in createTransaction: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		// Log the request body for debugging
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			err = fmt.Errorf("error reading request: %v", err)
			return
		}
		r.Body.Close()
		
		// Create a new reader with the same bytes for json.Decode
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		log.Printf("Received transaction request: %s", string(bodyBytes))
		
	var tx struct {
			From  string `json:"from"`
			To    string `json:"to"`
			Value uint64 `json:"value"`
			Data  string `json:"data,omitempty"`
		}
		
		if err = json.NewDecoder(r.Body).Decode(&tx); err != nil {
		log.Printf("Transaction decode error: %v", err)
			err = fmt.Errorf("invalid transaction format: %v", err)
		return
	}
	
	// Temel doğrulama kontrolleri
	if tx.From == "" {
			err = errors.New("sender address cannot be empty")
		return
	}
	
	if tx.To == "" {
			err = errors.New("recipient address cannot be empty")
		return
	}
	
	if tx.Value <= 0 {
			err = fmt.Errorf("invalid transaction amount: %d", tx.Value)
		return
	}
	
	// Debug logging
		log.Printf("Creating transaction: From=%s, To=%s, Value=%d", tx.From, tx.To, tx.Value)
		
		// Ayrıca kullanıcının bekleyen diğer işlemlerini de kontrol edelim
		pendingTxs := ws.blockchain.GetPendingTransactions()
		pendingSpend := uint64(0)
		
		for _, pendingTx := range pendingTxs {
			if pendingTx.From == tx.From {
				pendingSpend += pendingTx.Value
			}
		}
		
		// Get sender balance
		senderBalanceBigInt, err := ws.blockchain.GetBalance(tx.From)
	if err != nil {
			log.Printf("Error getting balance for sender %s: %v", tx.From, err)
			err = fmt.Errorf("cannot get sender balance: %v", err)
		return
	}

		// Check if sender balance can be represented as uint64
		if !senderBalanceBigInt.IsUint64() {
			// Balance is too large for uint64, assume sufficient for this check
			// Continue processing
		} else {
			senderBalance := senderBalanceBigInt.Uint64()
			
			// Toplam harcama = bekleyen harcamalar + yeni işlem
			totalSpend := pendingSpend + tx.Value
			
			if totalSpend > senderBalance {
				log.Printf("Insufficient balance for transaction: required=%d, available=%d, pending=%d, total=%d", 
					tx.Value, senderBalance, pendingSpend, totalSpend)
				err = fmt.Errorf("insufficient balance: required=%d, available=%d, pending=%d", 
					tx.Value, senderBalance, pendingSpend)
		return
			}
		}
		
		// Create a simple transaction
		simpleTransaction = &blockchain.Transaction{
			ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
			From:      tx.From,
			To:        tx.To,
			Value:     tx.Value,
			Timestamp: time.Now().Unix(),
			Type:      "regular",
		}
		
		// Data handling
	if tx.Data != "" {
			simpleTransaction.Data = []byte(tx.Data)
		}
		
		// Add transaction to pool
		if err = ws.blockchain.AddTransaction(simpleTransaction); err != nil {
			log.Printf("Error adding transaction to pool: %v", err)
		return
	}
	
		log.Printf("Transaction added to pool: %s", simpleTransaction.ID)
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

		// Return the transaction
	w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(simpleTransaction)
		
	case <-ctx.Done():
		log.Printf("Timeout creating transaction: %v", ctx.Err())
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	}
}

// createWallet handles the wallet creation endpoint
func (ws *WebServer) createWallet(w http.ResponseWriter, r *http.Request) {
	// Automatically handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var response struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
		
		Balance    uint64 `json:"balance"`
		Success    bool   `json:"success"`
	}
	var err error
	
	// Do the wallet creation in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in createWallet: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Starting wallet creation")
		
		// Create wallet
	wallet, err := blockchain.CreateWallet()
	if err != nil {
			log.Printf("Failed to create wallet: %v", err)
			err = fmt.Errorf("failed to create wallet: %v", err)
		return
	}
		
		log.Printf("Wallet created with address: %s", wallet.Address)
		
		// Prepare initial response
		response = struct {
			Address    string `json:"address"`
			PublicKey  string `json:"publicKey"`
			PrivateKey string `json:"privateKey"`
			
			Balance    uint64 `json:"balance"`
			Success    bool   `json:"success"`
		}{
			Address:    wallet.Address,
			PublicKey:  wallet.KeyPair.GetPublicKeyString(),
			PrivateKey: wallet.KeyPair.GetPrivateKeyString(),
			Balance:    0, // Start with 0 balance
			Success:    true,
	}
	
	// Save wallet's key pair to blockchain
	ws.blockchain.AddKeyPair(wallet.Address, wallet.KeyPair)
		log.Printf("Key pair added for address: %s", wallet.Address)

		// Create account with 0 initial balance
		initialBalance := big.NewInt(0)
		if err := ws.blockchain.CreateAccount(wallet.Address, initialBalance); err != nil {
			log.Printf("Warning: Error creating account: %v", err)
		} else {
			log.Printf("Account created with initial balance: 0 tokens")
			
			// Pre-cache the balance
			ws.balanceCache.Store(wallet.Address, initialBalance)
			ws.balanceCacheExpiry.Store(wallet.Address, time.Now().Add(60*time.Second))
		}
		
		// Save blockchain state to disk after creating a wallet
		ws.blockchain.SaveToDisk()
		log.Printf("Blockchain state saved to disk after wallet creation")
		
		log.Printf("Wallet created successfully in %v", time.Since(start))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error in wallet creation: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Return the wallet information
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
		
	case <-ctx.Done():
		log.Printf("Timeout creating wallet: %v", ctx.Err())
		http.Error(w, "Wallet creation timed out", http.StatusGatewayTimeout)
	}
}

// getWalletBalance handles the wallet balance endpoint
func (ws *WebServer) getWalletBalance(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get address from URL parameters
	vars := mux.Vars(r)
	address := vars["address"]
	
	// Quick validation
	if address == "" {
		w.WriteHeader(http.StatusOK) // Still return 200
		json.NewEncoder(w).Encode(map[string]interface{}{
			"address": "",
			"balance": "0", // Send as string
		})
		return
	}
	
	log.Printf("WALLET BALANCE REQUEST for address: %s", address)
	
	// ALWAYS respond with something - default is 0 tokens
	response := struct {
		Address string `json:"address"`
		Balance string `json:"balance"` // Changed to string
	}{
		Address: address,
		Balance: "0", // Default balance as string
	}
	
	// Try to get from cache first (fastest)
	if cachedValue, ok := ws.balanceCache.Load(address); ok {
		cachedBalance := cachedValue.(*big.Int)
		if cachedBalance != nil && cachedBalance.Sign() > 0 {
			// Got valid cached value
			response.Balance = cachedBalance.String() // Use String() method
			log.Printf("Returning cached balance for %s: %s in %v", 
				address, response.Balance, time.Since(startTime))
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	
	// Create a context with a very short timeout - frontend is timing out anyway
	// so we might as well respond quickly with a default value
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	resultChan := make(chan *big.Int, 1)
	
	// Do the blockchain lookup in the background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getWalletBalance for %s: %v", address, r)
			}
			done <- true
		}()
		
		log.Printf("Checking if address %s exists in blockchain", address)
		
		// Check if the address is known
		_, keyExists := ws.blockchain.GetKeyPair(address)
		
		// Get account balance if possible - only confirmed balance
	balance, err := ws.blockchain.GetBalance(address)
		
	if err != nil {
			log.Printf("Error getting balance for %s: %v", address, err)
		return
	}
	
		// Use default for nil/zero/negative balance
		if balance == nil || balance.Sign() <= 0 {
			// If key exists but balance is 0, still use default
			if keyExists {
				log.Printf("Address %s exists but has zero balance", address)
			}
			return
		}
		
		// Got valid balance, send the result
		resultChan <- balance
		
		// Cache the balance for 30 seconds
		ws.balanceCache.Store(address, balance)
		ws.balanceCacheExpiry.Store(address, time.Now().Add(30*time.Second))
		
		log.Printf("Retrieved balance for %s: %s in %v", 
			address, balance.String(), time.Since(startTime))
	}()
	
	// Wait for either completion, result, or timeout
	select {
	case result := <-resultChan:
		// Got a balance result
		response.Balance = result.String()
	case <-done:
		// Done but no result sent - using default
		log.Printf("No valid balance returned for %s, using default", address)
	case <-ctx.Done():
		// Timeout - using default
		log.Printf("Timeout getting balance for %s, using default", address)
	}
	
	// Always return OK with the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	
	log.Printf("Completed balance request for %s in %v (balance: %s)", 
		address, time.Since(startTime), response.Balance)
}

// mineBlock handles the mining endpoint
func (ws *WebServer) mineBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Validator string `json:"validator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding mining request: %v", err)
		http.Error(w, fmt.Sprintf("invalid request format: %v", err), http.StatusBadRequest)
		return
	}
	
	if req.Validator == "" {
		log.Printf("Mining request error: validator address is empty")
		http.Error(w, "validator address is required", http.StatusBadRequest)
		return
	}
	
	// Log the mining attempt
	log.Printf("Mining attempt from address: %s", req.Validator)
	
	// Check if the address is a registered validator
	if !ws.blockchain.IsValidator(req.Validator) {
		log.Printf("Unauthorized mining attempt from non-validator address: %s", req.Validator)
		http.Error(w, fmt.Sprintf("address %s is not a registered validator", req.Validator), http.StatusUnauthorized)
		return
	}
	
	// Get validator's key pair
	validators := ws.blockchain.GetValidators()
	if len(validators) == 0 {
		log.Printf("No validators found in the blockchain")
		http.Error(w, "no validators available", http.StatusBadRequest)
		return
	}

	validatorAddress := validators[0].Address
	keyPair, exists := ws.blockchain.GetKeyPair(validatorAddress)
	if !exists {
		log.Printf("Key pair not found for validator: %s", validatorAddress)
		
		// List all available addresses for debugging
		addresses := ws.blockchain.GetAllAddresses()
		log.Printf("Available addresses in blockchain: %v", addresses)
		
		http.Error(w, fmt.Sprintf("validator's key pair not found for %s", validatorAddress), http.StatusBadRequest)
		return
	}
	log.Printf("Retrieved key pair for validator: %s", validatorAddress)
	
	// Get validator's human proof
	humanProof := ws.blockchain.GetHumanProof(req.Validator)
	if humanProof == "" {
		log.Printf("Human proof not found for validator: %s", req.Validator)
		http.Error(w, "validator's human proof not found", http.StatusBadRequest)
		return
	}
	log.Printf("Retrieved human proof for validator: %s", req.Validator)
	
	// Get pending transactions
	pendingTxs := ws.blockchain.GetPendingTransactions()
	log.Printf("Retrieved %d pending transactions", len(pendingTxs))
	
	if len(pendingTxs) == 0 {
		log.Printf("No pending transactions to mine for validator: %s", req.Validator)
		http.Error(w, "no pending transactions to mine", http.StatusBadRequest)
		return
	}
	
	// Validate and pre-process all transactions
	validTxs := []*blockchain.Transaction{}
	invalidTxs := []*blockchain.Transaction{}
	
	// Her bir göndericinin bloktaki tüm işlemler sonrası toplam harcamasını takip edelim
	senderSpends := make(map[string]uint64)
	senderBalances := make(map[string]uint64)
	
	for _, tx := range pendingTxs {
		// Validate transaction basics
		if tx.From == "" || tx.To == "" || tx.Value <= 0 {
			log.Printf("Invalid transaction found: From=%s, To=%s, Value=%d", tx.From, tx.To, tx.Value)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Get sender balance if not already cached
		if _, exists := senderBalances[tx.From]; !exists {
			balance, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Warning: Cannot get balance for sender %s: %v", tx.From, err)
			invalidTxs = append(invalidTxs, tx)
			continue
			}
			senderBalances[tx.From] = balance.Uint64()
		}
		
		// Calculate total spend for this sender so far
		if _, exists := senderSpends[tx.From]; !exists {
			senderSpends[tx.From] = 0
		}
		
		totalSpentBySender := senderSpends[tx.From] + tx.Value
		
		// Check if sender has enough balance considering all transactions in this block
		senderBalance := senderBalances[tx.From]
		
		if totalSpentBySender > senderBalance {
			log.Printf("Warning: Insufficient balance for transaction %s after considering previous txs in block (sender: %s, amount: %d, balance: %d, total spent: %d)",
				tx.ID, tx.From, tx.Value, senderBalance, totalSpentBySender)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Verify transaction signature
		if !tx.SimpleVerify() {
			log.Printf("Warning: Transaction %s has invalid signature", tx.ID)
			invalidTxs = append(invalidTxs, tx)
			continue
		}
		
		// Bu işlem geçerli, toplam harcamayı güncelleyelim
		senderSpends[tx.From] = totalSpentBySender
		validTxs = append(validTxs, tx)
		log.Printf("Valid transaction found: ID=%s, From=%s, To=%s, Value=%d", 
			tx.ID, tx.From, tx.To, tx.Value)
	}
	
	if len(validTxs) == 0 {
		log.Printf("No valid transactions to mine for validator: %s", req.Validator)
		http.Error(w, "no valid transactions to mine", http.StatusBadRequest)
		return
	}
	
	log.Printf("Found %d valid transactions out of %d pending transactions", len(validTxs), len(pendingTxs))
	
	// Create a new block
	lastBlock := ws.blockchain.GetLatestBlock()
	
	// Log latest block details
	log.Printf("Latest block: Index=%d, Hash=%s", lastBlock.Index, lastBlock.Hash)
	
	newBlock := &blockchain.Block{
		Index:        lastBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: validTxs, // Only include valid transactions
		PrevHash:     lastBlock.Hash,
		Validator:    req.Validator,
		HumanProof:   humanProof, // Validatör için saklanan gerçek human proof kullanıyoruz
	}
	
	// Calculate and set the block hash
	newBlock.Hash = newBlock.CalculateHash()
	log.Printf("New block created with hash: %s", newBlock.Hash)
	
	// Sign the block
	if err := newBlock.Sign(keyPair.PrivateKey); err != nil {
		log.Printf("Error during block signing by validator %s: %v", req.Validator, err)
		http.Error(w, fmt.Sprintf("block signing failed: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Block successfully signed by validator %s", req.Validator)
	
	// Add block to blockchain
	if err := ws.blockchain.AddBlock(newBlock); err != nil {
		log.Printf("Error adding block to blockchain: %v", err)
		http.Error(w, fmt.Sprintf("failed to add block: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Block #%d successfully added to blockchain", newBlock.Index)
	
	// Remove invalid transactions from the pool
	for _, tx := range invalidTxs {
		if err := ws.blockchain.RemoveTransaction(tx.ID); err != nil {
			log.Printf("Warning: Failed to remove invalid transaction %s: %v", tx.ID, err)
		}
	}
	
	// Process valid transactions and update balances
	successfulTxs := []*blockchain.Transaction{}
	failedTxs := []*blockchain.Transaction{}
	
	for _, tx := range validTxs {
		// Final balance check before updating
		currentBalance, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Error checking balance for %s: %v", tx.From, err)
			failedTxs = append(failedTxs, tx)
			continue
		}
		
		if currentBalance.Cmp(big.NewInt(int64(tx.Value))) < 0 {
			log.Printf("Final balance check failed for tx %s: required=%d, available=%d", 
				tx.ID, tx.Value, currentBalance)
			failedTxs = append(failedTxs, tx)
			continue
		}
		
		// Update balances
		if err := ws.blockchain.UpdateBalances(tx); err != nil {
			log.Printf("Failed to update balances for transaction %s: %v", tx.ID, err)
			failedTxs = append(failedTxs, tx)
		} else {
			// Transaction successfully processed
			successfulTxs = append(successfulTxs, tx)
			log.Printf("Successfully processed transaction %s: %d tokens from %s to %s",
				tx.ID, tx.Value, tx.From, tx.To)
				
			// Get and log new balances for verification
			newSenderBalance, _ := ws.blockchain.GetBalance(tx.From)
			newReceiverBalance, _ := ws.blockchain.GetBalance(tx.To)
			log.Printf("Updated balances - Sender %s: %d, Receiver %s: %d", 
				tx.From, newSenderBalance, tx.To, newReceiverBalance)
		}
	}
	
	// Log summary
	log.Printf("Block #%d mining summary: %d successful transactions, %d failed transactions",
		newBlock.Index, len(successfulTxs), len(failedTxs))
	
	// Return block information with successful transactions
	response := struct {
		Block             *blockchain.Block          `json:"block"`
		SuccessfulTxs     []*blockchain.Transaction  `json:"successfulTransactions"`
		FailedTxs         []*blockchain.Transaction  `json:"failedTransactions"`
		InvalidTxs        int                        `json:"invalidTransactions"`
	}{
		Block:         newBlock,
		SuccessfulTxs: successfulTxs,
		FailedTxs:     failedTxs,
		InvalidTxs:    len(invalidTxs),
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// registerValidator handles the validator registration endpoint
func (ws *WebServer) registerValidator(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Address    string `json:"address"`
		HumanProof string `json:"humanProof"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate inputs
	if req.Address == "" {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}
	
	if req.HumanProof == "" {
		http.Error(w, "humanProof is required", http.StatusBadRequest)
		return
	}
	
	// Check if address has a key pair
	if _, exists := ws.blockchain.GetKeyPair(req.Address); !exists {
		http.Error(w, "address does not have a registered key pair", http.StatusBadRequest)
		return
	}
	
	// Check if already a validator
	if ws.blockchain.IsValidator(req.Address) {
		http.Error(w, "address is already a validator", http.StatusConflict)
		return
	}
	
	// Add as validator
	if err := ws.blockchain.AddValidator(req.Address, req.HumanProof); err != nil {
		log.Printf("Failed to register validator: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Save blockchain state to disk after registering a validator
	go ws.blockchain.SaveToDisk()
	
	log.Printf("Address %s registered as validator with proof: %s", req.Address, req.HumanProof)
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Validator registered successfully",
		"address": req.Address,
	})
}

// getValidators returns the list of registered validators with caching
func (ws *WebServer) getValidators(w http.ResponseWriter, r *http.Request) {
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Statik validator listesi - ValidatorInfo struct'ının gerçek yapısına uygun
	// Sadece zorunlu alanlar olan Address ve HumanProof kullanılıyor
	defaultValidators := []blockchain.ValidatorInfo{}
	
	// İlk olarak önbellekteki verileri kontrol edelim (30 saniyeden daha yeni ise)
	ws.validatorsCacheMutex.RLock()
	cacheAge := time.Since(ws.validatorsCacheTime)
	hasCache := len(ws.validatorsCache) > 0 && cacheAge < 30*time.Second
	
	// Eğer önbellekte güncel veri varsa, hemen döndürelim
	if hasCache {
		validators := ws.validatorsCache // Kopyasını alalım
		ws.validatorsCacheMutex.RUnlock()
		
		log.Printf("Returning %d validators from cache (age: %v)", len(validators), cacheAge)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(validators)
		return
	}
	
	// Çok eski bile olsa herhangi bir önbellek verisi var mı?
	staleCacheExists := len(ws.validatorsCache) > 0
	staleValidators := ws.validatorsCache
	ws.validatorsCacheMutex.RUnlock()
	
	// Asenkron olarak validator listesini güncellemeye çalışalım
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background validator update: %v", r)
			}
		}()
		
		// 5 saniyelik kısa bir timeout ile deneyelim
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Kanallarla iletişim kuralım
		done := make(chan bool, 1)
		var validators []blockchain.ValidatorInfo
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in validator fetching: %v", r)
				}
				done <- true
			}()
			
			start := time.Now()
			log.Printf("Background fetching validators from blockchain")
			
			// Get validators from blockchain
			allValidators := ws.blockchain.GetValidators()
			log.Printf("Found %d total validators in blockchain", len(allValidators))
			
			// Filter only active validators
			validators = make([]blockchain.ValidatorInfo, 0)
			for _, v := range allValidators {
				// Check if validator is active by checking if they have mined any blocks
				hasMinedBlocks := false
				chainHeight := ws.blockchain.GetChainHeight()
				
				// Check last 10 blocks for this validator
				for i := uint64(0); i < 10 && i <= chainHeight; i++ {
					block, err := ws.blockchain.GetBlockByIndex(i)
					if err != nil {
						continue
					}
					if block.Validator == v.Address {
						hasMinedBlocks = true
						break
					}
				}
				
				if hasMinedBlocks {
					validators = append(validators, v)
					log.Printf("Found active validator: %s", v.Address)
				} else {
					log.Printf("Found inactive validator: %s", v.Address)
				}
			}
			
			if len(validators) > 0 {
				// Önbelleği güncelle
				ws.validatorsCacheMutex.Lock()
				ws.validatorsCache = validators
				ws.validatorsCacheTime = time.Now()
				ws.validatorsCacheMutex.Unlock()
				
				log.Printf("Background updated validator cache with %d active validators in %v", 
					len(validators), time.Since(start))
			} else {
				log.Printf("No active validators found in blockchain")
			}
		}()
		
		// Wait for either completion or timeout
		select {
		case <-done:
			// İşlem tamamlandı, önbellek güncellendi
		case <-ctx.Done():
			log.Printf("Background validator update timed out: %v", ctx.Err())
		}
	}()
	
	// Hemen yanıt verelim - Önce eski önbellek, yoksa varsayılan veri
	if staleCacheExists && len(staleValidators) > 0 {
		log.Printf("Returning %d validators from stale cache immediately", len(staleValidators))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(staleValidators)
		return
	}
	
	// Önbellekte hiç veri yoksa, boş liste döndür
	log.Printf("No validator cache available, returning empty list")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(defaultValidators)
}

// getConfirmedTransactions handles the confirmed transactions endpoint with caching
func (ws *WebServer) getConfirmedTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers and Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get limit parameter, default to 30
	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// Cap the limit to prevent performance issues
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}
	
	// Önbellekteki verileri kontrol edelim (15 saniyeden daha yeni ise)
	ws.confirmedTxCacheMutex.RLock()
	cacheAge := time.Since(ws.confirmedTxCacheTime)
	hasCache := len(ws.confirmedTxCache) > 0 && cacheAge < 15*time.Second
	
	// Eğer önbellekte güncel veri varsa ve istenen limit önbellek boyutundan az veya eşitse, hemen döndürelim
	if hasCache && limit <= len(ws.confirmedTxCache) {
		// Önbellekten limiti kadar veri alalım
		txs := ws.confirmedTxCache
		if limit < len(txs) {
			txs = txs[:limit]
		}
		ws.confirmedTxCacheMutex.RUnlock()
		
		log.Printf("Returning %d confirmed transactions from cache (age: %v)", len(txs), cacheAge)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(txs)
		return
	}
	ws.confirmedTxCacheMutex.RUnlock()
	
	// Önbellekte veri yoksa veya eski ise veya istenen limit önbellek boyutundan büyükse, yeni veri alalım
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second) // Kısa bir timeout kullanarak hızlı cevap dönelim
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var confirmedTxs []*blockchain.Transaction
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getConfirmedTransactions: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Getting confirmed transactions from blockchain, limit=%d", limit)
		
		// Initialize the result array
		confirmedTxs = make([]*blockchain.Transaction, 0)
		
		// Get blockchain height
		height := int(ws.blockchain.GetChainHeight())
		
		// Sadece son 10 bloğa bakalım
		maxBlocksToCheck := 10
		if height < maxBlocksToCheck {
			maxBlocksToCheck = height + 1
		}
		
		// En son limiti aşmamak için her bloktan az sayıda işlem alalım
		txsPerBlock := limit / maxBlocksToCheck
		if txsPerBlock < 5 {
			txsPerBlock = 5
		}
		
		// Onaylanmış işlemleri en son bloklardan alalım
		for i := height; i >= (height-maxBlocksToCheck+1) && i >= 0 && len(confirmedTxs) < limit; i-- {
			block, err := ws.blockchain.GetBlockByIndex(uint64(i))
			if err != nil {
				log.Printf("Error fetching block at index %d: %v", i, err)
				continue
			}
			
			// Her bloktan en son birkaç işlemi alalım
			txsToProcess := block.Transactions
			if len(txsToProcess) > txsPerBlock {
				txsToProcess = txsToProcess[len(txsToProcess)-txsPerBlock:]
			}
			
			for _, tx := range txsToProcess {
				// Skip coinbase/reward transactions
				if tx.From == "0" || tx.From == "" {
					continue
				}
				
				// Create a copy
				txCopy := *tx
				// Add status and block information
				txCopy.Status = "confirmed"
				txCopy.BlockIndex = int64(block.Index)
				txCopy.BlockHash = block.Hash
				
				confirmedTxs = append(confirmedTxs, &txCopy)
				
				if len(confirmedTxs) >= limit {
					break
				}
			}
		}
		
		// Önbelleği güncelle - tüm işlemleri saklayalım (limitle sınırlamadan)
		ws.confirmedTxCacheMutex.Lock()
		ws.confirmedTxCache = confirmedTxs
		ws.confirmedTxCacheTime = time.Now()
		ws.confirmedTxCacheMutex.Unlock()
		
		log.Printf("Retrieved %d confirmed transactions in %v", len(confirmedTxs), time.Since(start))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting confirmed transactions: %v", err)
			// Return an empty array instead of an error
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
			return
		}
		
		// Create empty array if nil
		if confirmedTxs == nil {
			confirmedTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Limit the response if needed
		if len(confirmedTxs) > limit {
			confirmedTxs = confirmedTxs[:limit]
		}
		
		// Return the transactions as JSON
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(confirmedTxs)
		if err != nil {
			log.Printf("Error encoding confirmed transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		
	case <-ctx.Done():
		log.Printf("Timeout getting confirmed transactions: %v", ctx.Err())
		
		// Önbellekte herhangi bir veri varsa, eski de olsa döndürelim
		ws.confirmedTxCacheMutex.RLock()
		hasCacheData := len(ws.confirmedTxCache) > 0
		cachedTxs := ws.confirmedTxCache // Kopyasını alalım
		if limit < len(cachedTxs) {
			cachedTxs = cachedTxs[:limit]
		}
		ws.confirmedTxCacheMutex.RUnlock()
		
		if hasCacheData {
			log.Printf("Returning %d confirmed transactions from stale cache due to timeout", len(cachedTxs))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cachedTxs)
			return
		}
		
		// Hiç önbellek verisi yoksa boş dizi döndür
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(make([]*blockchain.Transaction, 0))
	}
}

// importWallet handles the wallet import endpoint
func (ws *WebServer) importWallet(w http.ResponseWriter, r *http.Request) {
	// Automatically handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		PrivateKey string `json:"privateKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.PrivateKey == "" {
		http.Error(w, "Private key is required", http.StatusBadRequest)
		return
	}

	// Import crypto/rand to use in this function
	privKey, err := blockchain.ImportPrivateKey(req.PrivateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid private key: %v", err), http.StatusBadRequest)
		return
	}

	// Use the private key from the blockchain package
	keyPair := &blockchain.KeyPair{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
	}

	// Generate address from public key
	address := blockchain.GenerateAddress(keyPair.PublicKey)

	// Check if wallet already exists in blockchain
	existingKeyPair, exists := ws.blockchain.GetKeyPair(address)
	if !exists {
		// Save wallet's key pair to blockchain
		ws.blockchain.AddKeyPair(address, keyPair)

		// Check if account exists, if not create it with initial balance
		_, err = ws.blockchain.GetBalance(address)
		if err != nil {
			initialBalance := big.NewInt(0) // Start with 0 tokens
			err = ws.blockchain.CreateAccount(address, initialBalance)
			if err != nil {
				log.Printf("Error creating account during import: %v", err)
			}
		}

		// Save blockchain state after import
		go ws.blockchain.SaveToDisk()
	} else {
		// Use existing key pair for consistent behavior
		keyPair = existingKeyPair
	}

	// Respond with wallet information
	response := struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
		Exists     bool   `json:"exists"`
	}{
		Address:    address,
		PublicKey:  keyPair.GetPublicKeyString(),
		PrivateKey: req.PrivateKey,
		Exists:     exists,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !exists {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

// Transfer handles the transfer endpoint
func (ws *WebServer) transfer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Handle preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Parse request body
	var req struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Value uint64 `json:"value"` // Changed from string to uint64
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error parsing transfer request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if req.From == "" || req.To == "" || req.Value == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Create transaction
	simpleTransaction := &blockchain.Transaction{
		ID:        uuid.New().String(),
		From:      req.From,
		To:        req.To,
		Value:     req.Value,
		Timestamp: time.Now().Unix(),
		Signature: []byte("system_transfer"), // Special system signature, ideally should be properly signed
		Type:      "regular",
		Status:    "pending",
	}
	
	// Create a context with timeout for the transfer operation
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	// Use a channel to get the result
	errCh := make(chan error, 1)
	
	go func() {
		// Add transaction to the blockchain
		err := ws.blockchain.AddTransaction(simpleTransaction)
		errCh <- err
	}()
	
	// Wait for the transaction to complete or timeout
	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("Transfer error: %v", err)
			http.Error(w, fmt.Sprintf("Transfer failed: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Success - transaction was added to the pool
		log.Printf("Transaction added to pool: %s", simpleTransaction.ID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(simpleTransaction)
		
	case <-ctx.Done():
		log.Printf("Timeout creating transaction: %v", ctx.Err())
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	}
}

// SignedRequest represents a request signed by an admin
type SignedRequest struct {
	Action     string            `json:"action"`
	Data       map[string]string `json:"data"`
	AdminAddress string          `json:"adminAddress"`
	Signature  string            `json:"signature"`
	Timestamp  int64             `json:"timestamp"`
}

// verifyAdminSignature verifies the admin signature on a request
func (ws *WebServer) verifyAdminSignature(req *types.SignedRequest) (bool, error) {
	// Verify timestamp (within 5 minutes)
	if time.Now().Unix()-req.Timestamp > 300 {
		return false, fmt.Errorf("request expired")
	}

	// Verify admin address
	if !ws.validatorManager.IsAdmin(req.AdminAddress) {
		return false, fmt.Errorf("invalid admin address")
	}

	// Verify signature
	valid, err := ws.validatorManager.VerifySignature(req)
	if err != nil {
		log.Printf("Error verifying signature: %v", err)
		return false, err
	}

	return valid, nil
}

// approveValidator handles approving a validator
func (ws *WebServer) approveValidator(w http.ResponseWriter, r *http.Request) {
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify admin signature
	if valid, err := ws.verifyAdminSignature(&req); !valid {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Extract validator address from request data
	validatorAddress, ok := req.Data["address"]
	if !ok {
		http.Error(w, "Missing validator address in request data", http.StatusBadRequest)
		return
	}

	// Approve the validator
	if err := ws.validatorManager.ApproveValidator(req.AdminAddress, validatorAddress); err != nil {
		http.Error(w, fmt.Sprintf("Failed to approve validator: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Validator %s approved successfully", validatorAddress),
	})
}

// rejectValidator handles rejecting a validator
func (ws *WebServer) rejectValidator(w http.ResponseWriter, r *http.Request) {
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify admin signature
	if valid, err := ws.verifyAdminSignature(&req); !valid {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Extract validator address and reason from request data
	validatorAddress, ok := req.Data["address"]
	if !ok {
		http.Error(w, "Missing validator address in request data", http.StatusBadRequest)
		return
	}

	reason := req.Data["reason"]
	if reason == "" {
		reason = "No reason provided"
	}

	// Reject the validator
	if err := ws.validatorManager.RejectValidator(validatorAddress, req.AdminAddress, reason); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reject validator: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Validator %s rejected successfully", validatorAddress),
	})
}

// suspendValidator handles suspending a validator
func (ws *WebServer) suspendValidator(w http.ResponseWriter, r *http.Request) {
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify admin signature
	if valid, err := ws.verifyAdminSignature(&req); !valid {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Extract validator address and reason from request data
	validatorAddress, ok := req.Data["address"]
	if !ok {
		http.Error(w, "Missing validator address in request data", http.StatusBadRequest)
		return
	}

	reason := req.Data["reason"]
	if reason == "" {
		reason = "No reason provided"
	}

	// Suspend the validator
	if err := ws.validatorManager.SuspendValidator(req.AdminAddress, validatorAddress, reason); err != nil {
		http.Error(w, fmt.Sprintf("Failed to suspend validator: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Validator %s suspended successfully", validatorAddress),
	})
}

// addAdmin handles adding a new admin
func (ws *WebServer) addAdmin(w http.ResponseWriter, r *http.Request) {
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify admin signature
	if valid, err := ws.verifyAdminSignature(&req); !valid {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Extract new admin address from request data
	newAdminAddress, ok := req.Data["address"]
	if !ok {
		http.Error(w, "Missing address in request data", http.StatusBadRequest)
		return
	}

	// Add the new admin
	if err := ws.validatorManager.AddAdmin(newAdminAddress, req.AdminAddress); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add admin: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Admin %s added successfully", newAdminAddress),
	})
}

// removeAdmin handles removing an admin
func (ws *WebServer) removeAdmin(w http.ResponseWriter, r *http.Request) {
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify admin signature
	if valid, err := ws.verifyAdminSignature(&req); !valid {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Extract admin address to remove from request data
	adminToRemove, ok := req.Data["address"]
	if !ok {
		http.Error(w, "Missing address in request data", http.StatusBadRequest)
		return
	}

	// Remove the admin
	if err := ws.validatorManager.RemoveAdmin(adminToRemove, req.AdminAddress); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove admin: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Admin %s removed successfully", adminToRemove),
	})
}

// listAdmins returns the list of current admins
func (ws *WebServer) listAdmins(w http.ResponseWriter, r *http.Request) {
	// Get the list of admins
	admins := ws.validatorManager.GetAdmins()
	
	// Return the list
	response := map[string]interface{}{
		"success": true,
		"admins":  admins,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listProposals returns the list of governance proposals
func (ws *WebServer) listProposals(w http.ResponseWriter, r *http.Request) {
	if ws.governance == nil {
		http.Error(w, "Governance system not enabled", http.StatusServiceUnavailable)
		return
	}
	
	// Get status filter from query parameters
	statusParam := r.URL.Query().Get("status")
	
	var proposals []*consensus.Proposal
	if statusParam != "" {
		status := consensus.ProposalStatus(statusParam)
		proposals = ws.governance.ListProposals(status)
	} else {
		proposals = ws.governance.ListProposals()
	}
	
	// Return the list
	response := map[string]interface{}{
		"success":   true,
		"proposals": proposals,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getProposal returns details of a specific proposal
func (ws *WebServer) getProposal(w http.ResponseWriter, r *http.Request) {
	if ws.governance == nil {
		http.Error(w, "Governance system not enabled", http.StatusServiceUnavailable)
		return
	}
	
	// Get proposal ID from URL parameters
	vars := mux.Vars(r)
	proposalID := vars["id"]
	
	// Get the proposal
	proposal, err := ws.governance.GetProposal(proposalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get proposal: %v", err), http.StatusNotFound)
		return
	}
	
	// Return the proposal
	response := map[string]interface{}{
		"success":  true,
		"proposal": proposal,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ProposalRequest represents a request to create a new proposal
type ProposalRequest struct {
	Creator     string            `json:"creator"`
	Type        string            `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Data        map[string]string `json:"data"`
	Signature   string            `json:"signature"`
}

// createProposal creates a new governance proposal
func (ws *WebServer) createProposal(w http.ResponseWriter, r *http.Request) {
	if ws.governance == nil {
		http.Error(w, "Governance system not enabled", http.StatusServiceUnavailable)
		return
	}
	
	// Decode request
	var req ProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request format: %v", err), http.StatusBadRequest)
		return
	}
	
	// Verify signature (in a real system)
	// For development, we'll skip this
	
	// Create the proposal
	proposalID, err := ws.governance.CreateProposal(
		req.Creator,
		consensus.ProposalType(req.Type),
		req.Title,
		req.Description,
		req.Data,
	)
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create proposal: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success":    true,
		"proposalID": proposalID,
		"message":    "Proposal created successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// VoteRequest represents a vote on a proposal
type VoteRequest struct {
	Voter      string `json:"voter"`
	ProposalID string `json:"proposalId"`
	InFavor    bool   `json:"inFavor"`
	Signature  string `json:"signature"`
}

// castVote casts a vote on a governance proposal
func (ws *WebServer) castVote(w http.ResponseWriter, r *http.Request) {
	if ws.governance == nil {
		http.Error(w, "Governance system not enabled", http.StatusServiceUnavailable)
		return
	}
	
	// Decode request
	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request format: %v", err), http.StatusBadRequest)
		return
	}
	
	// Verify signature (in a real system)
	// For development, we'll skip this
	
	// Cast the vote
	err := ws.governance.CastVote(req.ProposalID, req.Voter, req.InFavor)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cast vote: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Vote cast successfully on proposal %s", req.ProposalID),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getBlockByIndex handles retrieving a specific block by its index
func (ws *WebServer) getBlockByIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	vars := mux.Vars(r)
	indexStr := vars["index"]
	
	// Convert index string to int first (safer than directly to uint64)
	indexInt, err := strconv.Atoi(indexStr)
	if err != nil || indexInt < 0 {
		log.Printf("Error parsing block index: %v or negative index: %d", err, indexInt)
		http.Error(w, "invalid block index", http.StatusBadRequest)
		return
	}
	
	// Get chain height and validate that the requested index is in range
	chainHeight := int(ws.blockchain.GetChainHeight())
	if indexInt > chainHeight {
		log.Printf("Block index out of range: %d (max: %d)", indexInt, chainHeight)
		http.Error(w, fmt.Sprintf("block index out of range (max: %d)", chainHeight), http.StatusNotFound)
		return
	}
	
	// Convert to uint64 only after validation
	index := uint64(indexInt)
	
	// Check if we have a cached block
	indexKey := fmt.Sprintf("block_%d", index)
	if cachedValue, ok := ws.blockCache.Load(indexKey); ok {
		if expiryTime, ok := ws.blockCacheExpiry.Load(indexKey); ok {
			if time.Now().Before(expiryTime.(time.Time)) {
				// Cached value is still valid
				cachedBlock := cachedValue.(*blockchain.Block)
				
				log.Printf("Returning cached block for index %d", index)
				
				// Return the cached block with capitalized field names for React
				returnBlockWithCapitalizedFields(w, cachedBlock)
				return
			}
		}
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var block *blockchain.Block
	var blockErr error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getBlockByIndex for %d: %v", index, r)
				blockErr = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Block request for index: %d", index)
		
		// Get the block from blockchain
		block, blockErr = ws.blockchain.GetBlockByIndex(index)
		
		if blockErr == nil && block != nil {
			// Ensure block has a valid hash
			if block.Hash == "" {
				block.Hash = fmt.Sprintf("block_%d_%d", block.Index, block.Timestamp)
			}
			
			// Cache the block for 60 seconds - blocks don't change once created
			ws.blockCache.Store(indexKey, block)
			ws.blockCacheExpiry.Store(indexKey, time.Now().Add(60*time.Second))
			
			log.Printf("Retrieved block for index %d in %v", index, time.Since(start))
		} else {
			log.Printf("Error retrieving block at index %d: %v", index, blockErr)
		}
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if blockErr != nil {
			log.Printf("Error retrieving block at index %d: %v", index, blockErr)
			http.Error(w, fmt.Sprintf("block not found at index %d", index), http.StatusNotFound)
			return
		}
		
		if block == nil {
			log.Printf("No block found at index %d", index)
			http.Error(w, fmt.Sprintf("block not found at index %d", index), http.StatusNotFound)
			return
		}
		
		// Return the block with capitalized field names for React
		returnBlockWithCapitalizedFields(w, block)
		
	case <-ctx.Done():
		log.Printf("Timeout getting block at index %d: %v", index, ctx.Err())
		
		// Check for an expired cached value that we can use as a fallback
		if cachedValue, ok := ws.blockCache.Load(indexKey); ok {
			cachedBlock := cachedValue.(*blockchain.Block)
			
			log.Printf("Timeout - returning stale cached block for index %d", index)
			
			// Return the stale cached block with capitalized field names
			returnBlockWithCapitalizedFields(w, cachedBlock)
			return
		}
		
		// No cached value available
		http.Error(w, fmt.Sprintf("timeout retrieving block at index %d", index), http.StatusGatewayTimeout)
	}
}

// Helper function to return block with capitalized field names for React
func returnBlockWithCapitalizedFields(w http.ResponseWriter, block *blockchain.Block) {
	// Define a struct with capitalized field names
	type TransactionResponse struct {
		ID        string `json:"ID"`
		From      string `json:"From"`
		To        string `json:"To"`
		Value     uint64 `json:"Value"`
		Timestamp int64  `json:"Timestamp"`
		Status    string `json:"Status"`
		Type      string `json:"Type"`
		Data      []byte `json:"Data,omitempty"`
	}
	
	type BlockResponse struct {
		Index        uint64               `json:"Index"`
		Timestamp    int64                `json:"Timestamp"`
		Hash         string               `json:"Hash"`
		PrevHash     string               `json:"PrevHash"`
		Validator    string               `json:"Validator"`
		HumanProof   string               `json:"HumanProof"`
		Signature    []byte               `json:"Signature"`
		Reward       uint64               `json:"Reward"`
		Transactions []TransactionResponse `json:"Transactions"`
	}
	
	// Ensure block has a valid hash
	if block.Hash == "" {
		block.Hash = fmt.Sprintf("block_%d_%d", block.Index, block.Timestamp)
	}
	
	// Convert transactions to properly capitalized format
	txResponses := make([]TransactionResponse, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		status := "confirmed"
		if tx.Status != "" {
			status = tx.Status
		}
		
		txResponse := TransactionResponse{
			ID:        tx.ID,
			From:      tx.From,
			To:        tx.To,
			Value:     tx.Value,
			Timestamp: tx.Timestamp,
			Status:    status,
			Type:      tx.Type,
			Data:      tx.Data,
		}
		txResponses = append(txResponses, txResponse)
	}
	
	// Create final response
	blockResponse := BlockResponse{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Hash:         block.Hash,
		PrevHash:     block.PrevHash,
		Validator:    block.Validator,
		HumanProof:   block.HumanProof,
		Signature:    block.Signature,
		Reward:       block.Reward,
		Transactions: txResponses,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(blockResponse)
}

// getAllTransactions combines pending and confirmed transactions
func (ws *WebServer) getAllTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Parse limit from query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 30 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	// Cap limit at 100
	if limit > 100 {
		limit = 100
	}
	
	// Set a timeout for the handler - 30 seconds should be enough for all transactions
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	allTxs := make([]*blockchain.Transaction, 0)
	var err error
	
	// Do the work in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getAllTransactions: %v", r)
				err = fmt.Errorf("internal error: %v", r)
			}
			done <- true
		}()
		
		start := time.Now()
		log.Printf("Getting all transactions from blockchain, limit=%d", limit)
		
		// Initialize the result array
		allTxs = make([]*blockchain.Transaction, 0, limit)
		
		// First prioritize pending transactions - a quarter of the limit
		pendingLimit := limit / 4
		pendingStart := time.Now()
		
		// Check the cache first
		var pendingTxs []*blockchain.Transaction
		
		ws.pendingTxCacheMutex.RLock()
		if time.Since(ws.pendingTxCacheTime) < 30*time.Second {
			pendingTxs = ws.pendingTxCache
			log.Printf("Using cached pending transactions (%d items)", len(pendingTxs))
		}
		ws.pendingTxCacheMutex.RUnlock()
		
		// If no valid cache, get from blockchain
		if len(pendingTxs) == 0 {
			pendingTxs = ws.blockchain.GetPendingTransactions()
			
			// Update cache
			if len(pendingTxs) > 0 {
				ws.pendingTxCacheMutex.Lock()
				ws.pendingTxCache = pendingTxs
				ws.pendingTxCacheTime = time.Now()
				ws.pendingTxCacheMutex.Unlock()
			}
		}
		
		// Limit the number of pending transactions we process
		if len(pendingTxs) > pendingLimit {
			pendingTxs = pendingTxs[len(pendingTxs)-pendingLimit:]
		}
		
		for _, tx := range pendingTxs {
			// Create a copy
			txCopy := *tx
			// Add a status field for pending transactions
			txCopy.Status = "pending"
			allTxs = append(allTxs, &txCopy)
		}
		
		log.Printf("Got %d pending transactions in %v", len(pendingTxs), time.Since(pendingStart))
		
		// Calculate how many confirmed transactions we need
		remainingLimit := limit - len(allTxs)
		
		// Only get confirmed transactions if we haven't reached the limit
		if remainingLimit > 0 {
			// Get confirmed transactions - check only the latest 10 blocks
			maxBlocks := 10 // Limit to 10 blocks for performance
			confirmedStart := time.Now()
			
			// Get the current height (safely as int)
			chainHeight := int(ws.blockchain.GetChainHeight())
			
			// Loop through recent blocks to get transactions
			for i := chainHeight; i >= 0 && i > chainHeight-maxBlocks && len(allTxs) < limit; i-- {
				// Convert to uint64 only after validation
				blockIndex := uint64(i)
				
				// Try cache first
				blockKey := fmt.Sprintf("block_%d", blockIndex)
				var block *blockchain.Block
				
				if cachedValue, ok := ws.blockCache.Load(blockKey); ok {
					if expiryTime, ok := ws.blockCacheExpiry.Load(blockKey); ok {
						if time.Now().Before(expiryTime.(time.Time)) {
							block = cachedValue.(*blockchain.Block)
						}
					}
				}
				
				// If not in cache, get from blockchain
				if block == nil {
					var err error
					block, err = ws.blockchain.GetBlockByIndex(blockIndex)
					if err != nil {
						log.Printf("Error getting block at index %d: %v", i, err)
						continue
					}
					
					// Cache it
					ws.blockCache.Store(blockKey, block)
					ws.blockCacheExpiry.Store(blockKey, time.Now().Add(60*time.Second))
				}
				
				// Process transactions in the block
				blockTxCount := 0
				for _, tx := range block.Transactions {
					if len(allTxs) >= limit {
						break
					}
					
					// Only process a reasonable number of transactions per block
					if blockTxCount >= 20 {
						break
					}
					
					// Create a copy
					txCopy := *tx
					// Add status for confirmed transactions
					txCopy.Status = "confirmed"
					// Add block information - convert uint64 to int64
					txCopy.BlockIndex = int64(block.Index)
					txCopy.BlockHash = block.Hash
					
					allTxs = append(allTxs, &txCopy)
					blockTxCount++
				}
				
				log.Printf("Processed %d transactions from block %d", blockTxCount, i)
			}
			
			log.Printf("Got %d confirmed transactions in %v", len(allTxs)-len(pendingTxs), time.Since(confirmedStart))
		}
		
		log.Printf("Total transactions: %d (limit: %d) in %v", len(allTxs), limit, time.Since(start))
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		// Check for errors
		if err != nil {
			log.Printf("Error getting transactions: %v", err)
			// Still return what we have instead of an error
		}
		
		// Return transactions as JSON
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(allTxs)
		
	case <-ctx.Done():
		log.Printf("Timeout getting all transactions: %v", ctx.Err())
		
		// Return what we have so far
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(allTxs)
	}
}

// getWalletBalanceSimple is a simplified version of getWalletBalance
func (ws *WebServer) getWalletBalanceSimple(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Get address from URL parameters
	vars := mux.Vars(r)
	address := vars["address"]
	
	// Validate address
	if address == "" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"address": "",
			"balance": "0", // String representation
		})
		return
	}
	
	log.Printf("FAST BALANCE REQUEST for %s", address)
	
	// Default response - always return something valid
	response := struct {
		Address string `json:"address"`
		Balance string `json:"balance"` // String representation
	}{
		Address: address,
		Balance: "0", // Default balance as string
	}
	
	// Try to get from cache first (fastest)
	if cachedValue, ok := ws.balanceCache.Load(address); ok {
		cachedBalance := cachedValue.(*big.Int)
		if cachedBalance != nil && cachedBalance.Sign() >= 0 {
			// Use string representation directly
			response.Balance = cachedBalance.String()
			log.Printf("Fast endpoint: Cached balance for %s: %s (in %v)", 
				address, response.Balance, time.Since(startTime))
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	
	// Super fast timeout for blockchain lookup
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	done := make(chan bool, 1)
	resultChan := make(chan *big.Int, 1)
	
	// Try quick blockchain lookup in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in getWalletBalanceSimple: %v", r)
			}
			done <- true
		}()
		
		// Quick balance check - only confirmed balance
		balance, err := ws.blockchain.GetBalance(address)
		if err != nil || balance == nil {
			// Just silently use default (0)
			return
		}
		
		// Cache result for future use
		ws.balanceCache.Store(address, balance)
		ws.balanceCacheExpiry.Store(address, time.Now().Add(30*time.Second))
		
		// Send balance back on channel
		resultChan <- balance
	}()
	
	// Wait for result or timeout
	select {
	case result := <-resultChan:
		// Got real balance
		response.Balance = result.String()
		log.Printf("Fast endpoint: Retrieved balance for %s: %s (in %v)",
			address, response.Balance, time.Since(startTime))
	case <-done:
		// No valid result
		log.Printf("Fast endpoint: No valid balance for %s, using default (0) (in %v)",
			address, time.Since(startTime))
	case <-ctx.Done():
		// Timeout
		log.Printf("Fast endpoint: Timeout getting balance for %s, using default (0) (in %v)",
			address, time.Since(startTime))
	}
	
	// Send response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getHealthCheck provides a super fast health status endpoint for frontend connection checks
func (ws *WebServer) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Set headers for CORS
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Return a simple health status with no blockchain operations
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time": time.Now().Unix(),
	})
}

// Multi-signature request types
type CreateMultiSigWalletRequest struct {
	Address       string   `json:"address"`
	Owners        []string `json:"owners"`
	RequiredSigs  int      `json:"requiredSigs"`
	AdminAddress  string   `json:"adminAddress"`
	Signature     string   `json:"signature"`
}

type CreateMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	From          string `json:"from"`
	To            string `json:"to"`
	Value         string `json:"value"`
	Data          []byte `json:"data,omitempty"`
	Type          string `json:"type"`
	Signature     string `json:"signature"`
}

type SignMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	TxID          string `json:"txID"`
	Signer        string `json:"signer"`
	Signature     string `json:"signature"`
}

type ExecuteMultiSigTransactionRequest struct {
	WalletAddress string `json:"walletAddress"`
	TxID          string `json:"txID"`
	Signature     string `json:"signature"`
}

// Multi-signature handlers
func (ws *WebServer) createMultiSigWallet(w http.ResponseWriter, r *http.Request) {
	var req CreateMultiSigWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify signature
	signedReq := &types.SignedRequest{
		Action:      "create_multisig_wallet",
		Data:        map[string]string{"address": req.Address},
		AdminAddress: req.AdminAddress,
		Signature:   req.Signature,
		Timestamp:   time.Now().Unix(),
	}
	
	valid, err := ws.verifyAdminSignature(signedReq)
	if !valid || err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	err = ws.blockchain.CreateMultiSigWallet(req.Address, req.Owners, req.RequiredSigs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (ws *WebServer) getMultiSigWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	wallet, err := ws.blockchain.GetMultiSigWallet(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(wallet)
}

func (ws *WebServer) createMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := ws.blockchain.CreateMultiSigTransaction(
		req.WalletAddress,
		req.From,
		req.To,
		req.Value,
		req.Data,
		req.Type,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tx)
}

func (ws *WebServer) signMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	var req SignMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := ws.blockchain.SignMultiSigTransaction(
		req.WalletAddress,
		req.TxID,
		req.Signer,
		req.Signature,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (ws *WebServer) executeMultiSigTransaction(w http.ResponseWriter, r *http.Request) {
	var req ExecuteMultiSigTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := ws.blockchain.ExecuteMultiSigTransaction(req.WalletAddress, req.TxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (ws *WebServer) getMultiSigTransactionStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletAddress := vars["walletAddress"]
	txID := vars["txID"]

	status, err := ws.blockchain.GetMultiSigTransactionStatus(walletAddress, txID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (ws *WebServer) getMultiSigPendingTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletAddress := vars["walletAddress"]

	txs, err := ws.blockchain.GetMultiSigPendingTransactions(walletAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(txs)
}

// ... existing code ...

// revertTransaction reverts a transaction by its hash
func (ws *WebServer) revertTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	// Verify admin signature
	var req types.SignedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	valid, err := ws.verifyAdminSignature(&req)
	if !valid || err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Find and revert the transaction
	err = ws.blockchain.RevertTransaction(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// ... existing code ...

// Stop gracefully shuts down the web server
func (ws *WebServer) Stop() error {
	if ws.server != nil {
		// Give the server 5 seconds to finish processing existing requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return ws.server.Shutdown(ctx)
	}
	return nil
}