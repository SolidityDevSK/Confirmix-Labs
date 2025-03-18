package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/user/poa-poh-hybrid/pkg/blockchain"
	"github.com/user/poa-poh-hybrid/pkg/consensus"
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
	ws.router.HandleFunc("/api/transactions", ws.getTransactions).Methods("GET", "OPTIONS")
	ws.router.HandleFunc("/api/transaction", ws.createTransaction).Methods("POST", "OPTIONS")

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

// getTransactions handles the transactions endpoint
func (ws *WebServer) getTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	txs := ws.blockchain.GetPendingTransactions(10) // Get up to 10 pending transactions
	json.NewEncoder(w).Encode(txs)
}

// createTransaction handles the transaction creation endpoint
func (ws *WebServer) createTransaction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tx struct {
		From  string  `json:"from"`
		To    string  `json:"to"`
		Value float64 `json:"value"`
		Data  []byte  `json:"data,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Create and add transaction to pending pool
	transaction := blockchain.NewTransaction(tx.From, tx.To, tx.Value, tx.Data)
	if err := ws.blockchain.AddTransaction(transaction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
} 