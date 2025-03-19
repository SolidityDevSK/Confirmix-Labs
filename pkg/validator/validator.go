package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// Block represents a blockchain block
type Block struct {
	Index        int64         `json:"Index"`
	Hash         string        `json:"Hash"`
	PrevHash     string        `json:"PrevHash"`
	Timestamp    int64         `json:"Timestamp"`
	Transactions []Transaction `json:"Transactions"`
	Validator    string        `json:"Validator"`
	HumanProof   string        `json:"HumanProof"`
	Signature    string        `json:"Signature,omitempty"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string  `json:"ID"`
	From      string  `json:"From"`
	To        string  `json:"To"`
	Value     float64 `json:"Value"`
	Data      string  `json:"Data,omitempty"`
	Timestamp int64   `json:"Timestamp"`
	Signature string  `json:"Signature,omitempty"`
}

// Validator represents a blockchain validator node
type Validator struct {
	apiBaseURL   string
	address      string
	isRunning    bool
	interval     time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	client       *http.Client
}

// NewValidator creates a new validator instance
func NewValidator(apiBaseURL string, address string, interval time.Duration) *Validator {
	return &Validator{
		apiBaseURL: apiBaseURL,
		address:    address,
		interval:   interval,
		stopChan:   make(chan struct{}),
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Start begins the validation process
func (v *Validator) Start() error {
	if v.isRunning {
		return fmt.Errorf("validator is already running")
	}

	log.Printf("Starting validator with address: %s", v.address)
	log.Printf("Connected to API at: %s", v.apiBaseURL)
	v.isRunning = true
	v.wg.Add(1)

	go v.validationLoop()

	return nil
}

// Stop halts the validation process
func (v *Validator) Stop() {
	if !v.isRunning {
		return
	}

	log.Println("Stopping validator...")
	close(v.stopChan)
	v.wg.Wait()
	v.isRunning = false
	log.Println("Validator stopped")
}

// validationLoop runs continuously to validate pending transactions
func (v *Validator) validationLoop() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v.validatePendingTransactions()
		case <-v.stopChan:
			return
		}
	}
}

// validatePendingTransactions processes pending transactions and creates a new block
func (v *Validator) validatePendingTransactions() {
	// Get pending transactions
	pendingTxs, err := v.getPendingTransactions()
	if err != nil {
		log.Printf("Error fetching pending transactions: %v", err)
		return
	}

	if len(pendingTxs) == 0 {
		log.Println("No pending transactions to validate")
		return // Nothing to validate
	}

	log.Printf("Found %d pending transactions to validate", len(pendingTxs))

	// Mine a new block
	err = v.mineBlock()
	if err != nil {
		log.Printf("Error mining block: %v", err)
		return
	}

	log.Printf("Successfully validated transactions and created a new block")
}

// getPendingTransactions retrieves pending transactions from the API
func (v *Validator) getPendingTransactions() ([]Transaction, error) {
	resp, err := v.client.Get(fmt.Sprintf("%s/transactions", v.apiBaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var transactions []Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions response: %w", err)
	}

	return transactions, nil
}

// mineBlock requests the API to mine a new block
func (v *Validator) mineBlock() error {
	// Create request body
	reqBody, err := json.Marshal(map[string]string{
		"validator": v.address,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request to mine endpoint
	resp, err := v.client.Post(
		fmt.Sprintf("%s/mine", v.apiBaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to send mine request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Try to read error message
		var errMsg struct {
			Error string `json:"error"`
		}
		
		respBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Mining response body: %s", string(respBody))
		
		if err := json.Unmarshal(respBody, &errMsg); err == nil && errMsg.Error != "" {
			return fmt.Errorf("mining failed: %s", errMsg.Error)
		}
		return fmt.Errorf("mining failed with status: %d", resp.StatusCode)
	}

	return nil
} 