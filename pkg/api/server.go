package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/blockchain"
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/consensus"
)

// WebServer represents the web server instance
type WebServer struct {
	blockchain      *blockchain.Blockchain
	consensusEngine *consensus.HybridConsensus
	port           int
	router         *mux.Router
}

// NewWebServer creates a new web server instance
func NewWebServer(bc *blockchain.Blockchain, ce *consensus.HybridConsensus, port int) *WebServer {
	ws := &WebServer{
		blockchain:      bc,
		consensusEngine: ce,
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
	// Enable CORS for all routes
	ws.router.Use(enableCORS)

	// API routes
	ws.router.HandleFunc("/api/status", ws.getStatus).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/blocks", ws.getBlocks).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/transactions", ws.getAllTransactions).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/transactions/pending", ws.getPendingTransactions).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/transactions/confirmed", ws.getConfirmedTransactions).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/transaction", ws.createTransaction).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/mine", ws.mineBlock).Methods("POST", "OPTIONS")
	
	// Wallet routes
	ws.router.HandleFunc("/api/wallet/create", ws.createWallet).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/wallet/balance/{address}", ws.getWalletBalance).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/wallet/import", ws.importWallet).Methods("POST", "OPTIONS")
	
	// Validator routes
	ws.router.HandleFunc("/api/validator/register", ws.registerValidator).Methods("POST", "OPTIONS")
	ws.router.HandleFunc("/api/validators", ws.getValidators).Methods("GET", "OPTIONS")

	// Serve static files
	fs := http.FileServer(http.Dir("web"))
	ws.router.PathPrefix("/").Handler(http.StripPrefix("/", fs))
}

// Start starts the web server
func (ws *WebServer) Start() error {
	addr := fmt.Sprintf(":%d", ws.port)
	log.Printf("Web server listening on %s", addr)
	return http.ListenAndServe(addr, ws.router)
}

// getStatus handles the status endpoint
func (ws *WebServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := struct {
		Height    uint64 `json:"height"`
		LastBlock string `json:"lastBlock"`
	}{
		Height:    ws.blockchain.GetChainHeight(),
		LastBlock: ws.blockchain.GetLatestBlock().Hash,
	}
	json.NewEncoder(w).Encode(status)
}

// getBlocks handles the blocks endpoint
func (ws *WebServer) getBlocks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	// Get blocks from index max(0, height-limit) to height
	height := int(ws.blockchain.GetChainHeight())
	start := height - limit + 1
	if start < 0 {
		start = 0
	}
	
	blocks := make([]*blockchain.Block, 0)
	for i := start; i <= height; i++ {
		block, err := ws.blockchain.GetBlockByIndex(uint64(i))
		if err == nil {
			blocks = append(blocks, block)
		}
	}
	
	json.NewEncoder(w).Encode(blocks)
}

// getPendingTransactions handles the pending transactions endpoint
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
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var txs []*blockchain.Transaction
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
		
		// Create a fresh copy of transactions to avoid references to internal slice
		// This avoids mutex issues if the slice is modified while we're returning it
		pending := ws.blockchain.GetPendingTransactions()
		
		txs = make([]*blockchain.Transaction, 0, len(pending))
		
		for _, tx := range pending {
			// Create a shallow copy of each transaction
			txCopy := *tx
			// Add status field for pending transactions
			txCopy.Status = "pending"
			// Add empty block info for pending transactions for consistency with confirmed ones
			txCopy.BlockIndex = 0
			txCopy.BlockHash = ""
			txs = append(txs, &txCopy)
		}
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting pending transactions: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Create empty array if nil
		if txs == nil {
			txs = make([]*blockchain.Transaction, 0)
		}
		
		// Return the transactions as JSON
		err = json.NewEncoder(w).Encode(txs)
		if err != nil {
			log.Printf("Error encoding pending transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		
	case <-ctx.Done():
		log.Printf("Timeout getting pending transactions: %v", ctx.Err())
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
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
			From  string  `json:"from"`
			To    string  `json:"to"`
			Value float64 `json:"value"`
			Data  string  `json:"data,omitempty"`
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
			err = fmt.Errorf("invalid transaction amount: %f", tx.Value)
			return
		}
		
		// Debug logging
		log.Printf("Creating transaction: From=%s, To=%s, Value=%f", tx.From, tx.To, tx.Value)
		
		// Bakiye kontrolü ekleyelim
		senderBalance, err := ws.blockchain.GetBalance(tx.From)
		if err != nil {
			log.Printf("Error getting balance for sender %s: %v", tx.From, err)
			err = fmt.Errorf("cannot get sender balance: %v", err)
			return
		}
		
		// Ayrıca kullanıcının bekleyen diğer işlemlerini de kontrol edelim
		pendingTxs := ws.blockchain.GetPendingTransactions()
		pendingSpend := 0.0
		
		for _, pendingTx := range pendingTxs {
			if pendingTx.From == tx.From {
				pendingSpend += pendingTx.Value
			}
		}
		
		// Toplam harcama = bekleyen harcamalar + yeni işlem
		totalSpend := pendingSpend + tx.Value
		
		if totalSpend > senderBalance {
			log.Printf("Insufficient balance for transaction: required=%f, available=%f, pending=%f, total=%f", 
				tx.Value, senderBalance, pendingSpend, totalSpend)
			err = fmt.Errorf("insufficient balance: required=%.2f, available=%.2f, pending=%.2f", 
				tx.Value, senderBalance, pendingSpend)
			return
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
		enableCORS(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	wallet, err := blockchain.CreateWallet()
	if err != nil {
		http.Error(w, "Failed to create wallet", http.StatusInternalServerError)
		return
	}

	// Save wallet's key pair to blockchain
	ws.blockchain.AddKeyPair(wallet.Address, wallet.KeyPair)

	// Add initial balance to the wallet
	err = ws.blockchain.CreateAccount(wallet.Address, 1000.0)
	if err != nil {
		log.Printf("Error creating account: %v", err)
	}
	
	// Save blockchain state to disk after creating a wallet
	go ws.blockchain.SaveToDisk()

	// Respond with the wallet address and keys
	response := struct {
		Address    string `json:"address"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	}{
		Address:    wallet.Address,
		PublicKey:  wallet.KeyPair.GetPublicKeyString(),
		PrivateKey: wallet.KeyPair.GetPrivateKeyString(),
	}

	w.Header().Set("Content-Type", "application/json")
	enableCORS(w)
	json.NewEncoder(w).Encode(response)
}

// getWalletBalance handles the wallet balance endpoint
func (ws *WebServer) getWalletBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get address from URL parameters
	vars := mux.Vars(r)
	address := vars["address"]
	
	// Get account balance
	balance, err := ws.blockchain.GetBalance(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	// Return balance
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(balance)
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
	keyPair, exists := ws.blockchain.GetKeyPair(req.Validator)
	if !exists {
		log.Printf("Key pair not found for validator: %s", req.Validator)
		
		// List all available addresses for debugging
		addresses := ws.blockchain.GetAllAddresses()
		log.Printf("Available addresses in blockchain: %v", addresses)
		
		http.Error(w, fmt.Sprintf("validator's key pair not found for %s", req.Validator), http.StatusBadRequest)
		return
	}
	log.Printf("Retrieved key pair for validator: %s", req.Validator)
	
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
	senderSpends := make(map[string]float64)
	senderBalances := make(map[string]float64)
	
	for _, tx := range pendingTxs {
		// Validate transaction basics
		if tx.From == "" || tx.To == "" || tx.Value <= 0 {
			log.Printf("Invalid transaction found: From=%s, To=%s, Value=%f", tx.From, tx.To, tx.Value)
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
			senderBalances[tx.From] = balance
		}
		
		// Calculate total spend for this sender so far
		if _, exists := senderSpends[tx.From]; !exists {
			senderSpends[tx.From] = 0
		}
		
		totalSpentBySender := senderSpends[tx.From] + tx.Value
		
		// Check if sender has enough balance considering all transactions in this block
		if totalSpentBySender > senderBalances[tx.From] {
			log.Printf("Warning: Insufficient balance for transaction %s after considering previous txs in block (sender: %s, amount: %f, balance: %f, total spent: %f)",
				tx.ID, tx.From, tx.Value, senderBalances[tx.From], totalSpentBySender)
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
		log.Printf("Valid transaction found: ID=%s, From=%s, To=%s, Value=%f", 
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
		
		if currentBalance < tx.Value {
			log.Printf("Final balance check failed for tx %s: required=%f, available=%f", 
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
			log.Printf("Successfully processed transaction %s: %f tokens from %s to %s",
				tx.ID, tx.Value, tx.From, tx.To)
				
			// Get and log new balances for verification
			newSenderBalance, _ := ws.blockchain.GetBalance(tx.From)
			newReceiverBalance, _ := ws.blockchain.GetBalance(tx.To)
			log.Printf("Updated balances - Sender %s: %f, Receiver %s: %f", 
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

// getValidators returns the list of registered validators
func (ws *WebServer) getValidators(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	validators := ws.blockchain.GetValidators()
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(validators)
}

// getConfirmedTransactions handles the confirmed transactions endpoint
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
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
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
		
		// Get confirmed transactions from blocks
		confirmedTxs = make([]*blockchain.Transaction, 0)
		
		// Get optional limit parameter, default to 100
		limit := 100
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		
		// Get the blockchain height
		height := int(ws.blockchain.GetChainHeight())
		
		// Start from the latest block and work backwards
		count := 0
		for i := height; i >= 0 && count < limit; i-- {
			block, err := ws.blockchain.GetBlockByIndex(uint64(i))
			if err != nil {
				log.Printf("Error fetching block at index %d: %v", i, err)
				continue
			}
			
			// Add all transactions from this block
			for _, tx := range block.Transactions {
				// Skip coinbase/reward transactions if needed
				if tx.From == "0" || tx.From == "" {
					continue
				}
				
				// Create a copy to avoid reference issues
				txCopy := *tx
				// Add status and block information
				txCopy.Status = "confirmed"
				txCopy.BlockIndex = int64(block.Index)
				txCopy.BlockHash = block.Hash
				confirmedTxs = append(confirmedTxs, &txCopy)
				count++
				
				// Break if we've reached the limit
				if count >= limit {
					break
				}
			}
		}
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting confirmed transactions: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Create empty array if nil
		if confirmedTxs == nil {
			confirmedTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Return the transactions as JSON
		err = json.NewEncoder(w).Encode(confirmedTxs)
		if err != nil {
			log.Printf("Error encoding confirmed transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		
	case <-ctx.Done():
		log.Printf("Timeout getting confirmed transactions: %v", ctx.Err())
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	}
}

// getAllTransactions handles the all transactions endpoint
func (ws *WebServer) getAllTransactions(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers and Content-Type
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	
	// If it's an OPTIONS request, return immediately
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Set a timeout for the handler
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	
	// Use a done channel to signal when we're finished
	done := make(chan bool, 1)
	var allTxs []*blockchain.Transaction
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
		
		// Initialize the result array
		allTxs = make([]*blockchain.Transaction, 0)
		
		// Get limit parameter, default to 100
		limit := 100
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		
		// Step 1: Get pending transactions
		pending := ws.blockchain.GetPendingTransactions()
		for _, tx := range pending {
			// Create a copy
			txCopy := *tx
			// Add a status field for pending transactions
			txCopy.Status = "pending"
			allTxs = append(allTxs, &txCopy)
		}
		
		// Step 2: Get confirmed transactions (if we still have room in the limit)
		remainingLimit := limit - len(allTxs)
		if remainingLimit <= 0 {
			// We've already reached the limit with pending transactions
			return
		}
		
		// Get blockchain height
		height := int(ws.blockchain.GetChainHeight())
		
		// Get confirmed transactions from blocks
		confirmedCount := 0
		for i := height; i >= 0 && confirmedCount < remainingLimit; i-- {
			block, err := ws.blockchain.GetBlockByIndex(uint64(i))
			if err != nil {
				log.Printf("Error fetching block at index %d: %v", i, err)
				continue
			}
			
			for _, tx := range block.Transactions {
				// Skip coinbase/reward transactions if needed
				if tx.From == "0" || tx.From == "" {
					continue
				}
				
				// Create a copy
				txCopy := *tx
				// Add a status field for confirmed transactions
				txCopy.Status = "confirmed"
				// Add block info for confirmed transactions
				txCopy.BlockIndex = int64(block.Index)
				txCopy.BlockHash = block.Hash
				
				allTxs = append(allTxs, &txCopy)
				confirmedCount++
				
				if confirmedCount >= remainingLimit {
					break
				}
			}
		}
	}()
	
	// Wait for either completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error getting all transactions: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Create empty array if nil
		if allTxs == nil {
			allTxs = make([]*blockchain.Transaction, 0)
		}
		
		// Return the transactions as JSON
		err = json.NewEncoder(w).Encode(allTxs)
		if err != nil {
			log.Printf("Error encoding all transactions: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
		
	case <-ctx.Done():
		log.Printf("Timeout getting all transactions: %v", ctx.Err())
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	}
}

// importWallet handles the wallet import endpoint
func (ws *WebServer) importWallet(w http.ResponseWriter, r *http.Request) {
	// Automatically handle CORS preflight request
	if r.Method == "OPTIONS" {
		enableCORS(w)
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

	// Recreate private key from hex string
	privateKeyBytes, err := hex.DecodeString(req.PrivateKey)
	if err != nil {
		http.Error(w, "Invalid private key format", http.StatusBadRequest)
		return
	}

	curve := elliptic.P256()
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyBytes)

	// Create key pair
	keyPair := &blockchain.KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
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
			err = ws.blockchain.CreateAccount(address, 1000.0)
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
	enableCORS(w)
	if !exists {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
} 