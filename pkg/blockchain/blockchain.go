package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Blockchain represents the blockchain data structure
type Blockchain struct {
	Blocks         []*Block
	mu             sync.RWMutex
	pendingTxs     []*Transaction
	txPool         map[string]*Transaction
	validators     map[string]bool
	humanProofs    map[string]string // Map of validator address to human proof
	contractManager *ContractManager // Smart contract manager
	keyPairs       map[string]*KeyPair // Map of address to key pair
	accounts       map[string]*big.Int // Map of account addresses to balances (18 decimal precision)
	mutex          sync.RWMutex
}

// NewBlockchain creates a new blockchain with genesis block
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		Blocks:         make([]*Block, 0),
		pendingTxs:     make([]*Transaction, 0),
		txPool:         make(map[string]*Transaction),
		validators:     make(map[string]bool),
		humanProofs:    make(map[string]string),
		contractManager: NewContractManager(),
		keyPairs:       make(map[string]*KeyPair),
		accounts:       make(map[string]*big.Int),
	}

	// Try to load existing blockchain data
	loaded := bc.LoadFromDisk()
	
	// If no existing data, create genesis block
	if !loaded {
		genesisBlock := &Block{
			Index:        0,
			Timestamp:    time.Now().Unix(),
			Transactions: []*Transaction{},
			Hash:         "0",
			PrevHash:     "0",
			Validator:    "genesis",
			HumanProof:   "genesis",
		}
		
		bc.Blocks = append(bc.Blocks, genesisBlock)
		
		// Create genesis account with initial supply of 100 million ConX tokens (18 decimals)
		genesisAddress := "confirmix_genesis_address"
		
		// Create total supply as big.Int (100 million with 18 decimals)
		totalSupply := new(big.Int)
		totalSupply.SetString("100000000000000000000000000", 10) // 100 million tokens with 18 decimals
		bc.accounts[genesisAddress] = totalSupply
		
		// Save genesis block
		bc.SaveToDisk()
	}
	
	return bc
}

// GetBlockchainDataPath returns the path where blockchain data is stored
func GetBlockchainDataPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		homeDir = "."
	}
	
	dataDir := filepath.Join(homeDir, ".confirmix")
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Printf("Failed to create data directory: %v", err)
	}
	
	return dataDir
}

// SaveToDisk persists the blockchain state to disk
func (bc *Blockchain) SaveToDisk() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	dataDir := GetBlockchainDataPath()
	
	// Save blocks
	blocksFile := filepath.Join(dataDir, "blocks.json")
	blocksData, err := json.MarshalIndent(bc.Blocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blocks: %v", err)
	}
	
	if err := ioutil.WriteFile(blocksFile, blocksData, 0644); err != nil {
		return fmt.Errorf("failed to write blocks file: %v", err)
	}
	
	// Save validators
	validatorsMap := make(map[string]string)
	for addr := range bc.validators {
		validatorsMap[addr] = bc.humanProofs[addr]
	}
	
	validatorsFile := filepath.Join(dataDir, "validators.json")
	validatorsData, err := json.MarshalIndent(validatorsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validators: %v", err)
	}
	
	if err := ioutil.WriteFile(validatorsFile, validatorsData, 0644); err != nil {
		return fmt.Errorf("failed to write validators file: %v", err)
	}
	
	// Save accounts - ensure map uses string keys for JSON compatibility
	accountsMap := make(map[string]string)
	for addr, balance := range bc.accounts {
		accountsMap[addr] = balance.String() // big.Int has String() method
	}
	
	accountsFile := filepath.Join(dataDir, "accounts.json")
	accountsData, err := json.MarshalIndent(accountsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts: %v", err)
	}
	
	if err := ioutil.WriteFile(accountsFile, accountsData, 0644); err != nil {
		return fmt.Errorf("failed to write accounts file: %v", err)
	}
	
	log.Printf("Blockchain state saved to disk: %s", dataDir)
	return nil
}

// LoadFromDisk loads the blockchain state from disk
func (bc *Blockchain) LoadFromDisk() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	dataDir := GetBlockchainDataPath()
	
	// Load blocks
	blocksFile := filepath.Join(dataDir, "blocks.json")
	if _, err := os.Stat(blocksFile); os.IsNotExist(err) {
		log.Println("No existing blockchain data found")
		return false
	}
	
	blocksData, err := ioutil.ReadFile(blocksFile)
	if err != nil {
		log.Printf("Failed to read blocks file: %v", err)
		return false
	}
	
	var blocks []*Block
	if err := json.Unmarshal(blocksData, &blocks); err != nil {
		log.Printf("Failed to unmarshal blocks: %v", err)
		return false
	}
	
	bc.Blocks = blocks
	
	// Load validators
	validatorsFile := filepath.Join(dataDir, "validators.json")
	if _, err := os.Stat(validatorsFile); !os.IsNotExist(err) {
		validatorsData, err := ioutil.ReadFile(validatorsFile)
		if err == nil {
			var validatorsMap map[string]string
			if err := json.Unmarshal(validatorsData, &validatorsMap); err == nil {
				bc.validators = make(map[string]bool)
				bc.humanProofs = make(map[string]string)
				
				for addr, proof := range validatorsMap {
					bc.validators[addr] = true
					bc.humanProofs[addr] = proof
				}
			}
		}
	}
	
	// Load accounts
	accountsFile := filepath.Join(dataDir, "accounts.json")
	accountsData, err := ioutil.ReadFile(accountsFile)
	if err != nil {
		log.Printf("Failed to read accounts file: %v", err)
		return false
	}
	
	var accountsMap map[string]string
	if err := json.Unmarshal(accountsData, &accountsMap); err != nil {
		log.Printf("Failed to unmarshal accounts: %v", err)
		return false
	}
	
	bc.accounts = make(map[string]*big.Int)
	for addr, balanceStr := range accountsMap {
		balance := new(big.Int)
		success := false
		if balanceStr != "" {
			_, success = balance.SetString(balanceStr, 10)
		}
		
		if !success {
			log.Printf("Invalid balance format for %s: %s, setting to 0", addr, balanceStr)
			balance = big.NewInt(0)
		}
		
		bc.accounts[addr] = balance
		log.Printf("Loaded account %s with balance %s", addr, balance.String())
	}
	
	log.Printf("Blockchain state loaded from disk: %s", dataDir)
	log.Printf("Loaded %d blocks, %d pending transactions, %d accounts", 
		len(bc.Blocks), len(bc.txPool), len(bc.accounts))
	
	return true
}

// AddValidator adds a new authorized validator to the blockchain
func (bc *Blockchain) AddValidator(address string, humanProof string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if human proof is valid (in a real implementation, this would verify with PoH)
	if humanProof == "" {
		return errors.New("human proof is required for validators")
	}
	
	// Generate a new key pair for the validator
	keyPair, err := NewKeyPair()
	if err != nil {
		return err
	}
	
	bc.validators[address] = true
	bc.humanProofs[address] = humanProof
	bc.keyPairs[address] = keyPair
	return nil
}

// IsValidator checks if an address is an authorized validator
func (bc *Blockchain) IsValidator(address string) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.validators[address]
}

// GetHumanProof gets the human proof for a validator
func (bc *Blockchain) GetHumanProof(address string) string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.humanProofs[address]
}

// AddTransaction adds a new transaction to the transaction pool
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	// Basic validation checks
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}
	
	if tx.From == "" {
		return errors.New("sender address cannot be empty")
	}
	
	if tx.To == "" {
		return errors.New("recipient address cannot be empty")
	}
	
	// Value field is now a string representing big.Int
	if tx.Value == "" {
		return errors.New("transaction value cannot be empty")
	}
	
	// Parse value to ensure it's a valid big.Int
	txValue := new(big.Int)
	if _, success := txValue.SetString(tx.Value, 10); !success {
		return fmt.Errorf("invalid transaction amount: %s", tx.Value)
	}
	
	// Check if transaction value is positive
	if txValue.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("transaction amount must be positive: %s", tx.Value)
	}
	
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if transaction already exists
	if _, exists := bc.txPool[tx.ID]; exists {
		return errors.New("transaction already exists in the pool")
	}
	
	// Create recipient account if it doesn't exist
	if _, recipientExists := bc.accounts[tx.To]; !recipientExists {
		bc.accounts[tx.To] = big.NewInt(0)
	}
	
	// Special case for genesis funding transactions
	if tx.From == "confirmix_genesis_address" && tx.Signature == "genesis_funding" {
		// Allow these transactions without signature verification
		log.Printf("Genesis funding transaction to %s with amount %s", tx.To, tx.Value)
	} else {
		// Get sender balance
		senderBalance, exists := bc.accounts[tx.From]
		if !exists {
			return errors.New("sender account does not exist")
		}
		
		// Check if sender has sufficient balance
		if senderBalance.Cmp(txValue) < 0 {
			return fmt.Errorf("insufficient balance: have %s, need %s", senderBalance.String(), tx.Value)
		}
		
		// Verify transaction signature (except for special cases)
		if tx.Signature == "" {
			return errors.New("transaction must be signed")
		}
		
		// Skip signature verification for system transactions
		if !strings.HasPrefix(tx.Signature, "genesis_") && !strings.HasPrefix(tx.Signature, "system_") {
			// Would normally verify signature here
		}
	}
	
	// Add transaction to pool and pending transactions
	bc.txPool[tx.ID] = tx
	bc.pendingTxs = append(bc.pendingTxs, tx)
	
	log.Printf("Transaction added to pool: %s, from: %s, to: %s, amount: %s", 
		tx.ID, tx.From, tx.To, tx.Value)
	
	return nil
}

// GetPendingTransactions returns the list of pending transactions
func (bc *Blockchain) GetPendingTransactions() []*Transaction {
	bc.mu.RLock()
	
	transactionCount := len(bc.pendingTxs)
	
	// Hızlı bir kopya oluştur ve kilidi bırak
	result := make([]*Transaction, transactionCount)
	for i := 0; i < transactionCount && i < len(bc.pendingTxs); i++ {
		result[i] = bc.pendingTxs[i]
	}
	
	bc.mu.RUnlock()
	
	return result
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Verify block index
	if uint64(len(bc.Blocks)) != block.Index {
		return fmt.Errorf("invalid block index: expected %d, got %d", len(bc.Blocks), block.Index)
	}
	
	// Verify previous hash
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	if prevBlock.Hash != block.PrevHash {
		return fmt.Errorf("invalid previous hash: expected %s, got %s", prevBlock.Hash, block.PrevHash)
	}
	
	// Verify human proof
	if !bc.IsValidator(block.Validator) {
		return fmt.Errorf("invalid validator: %s is not an authorized validator", block.Validator)
	}
	
	// Verify that human proof matches
	expectedProof := bc.GetHumanProof(block.Validator)
	if expectedProof != block.HumanProof {
		return fmt.Errorf("invalid human proof: expected %s, got %s", expectedProof, block.HumanProof)
	}
	
	// Verify block signature
	err := bc.verifyBlockSignature(block)
	if err != nil {
		return fmt.Errorf("invalid block signature: %v", err)
	}
	
	// Add the block
	bc.Blocks = append(bc.Blocks, block)
	
	// Process all transactions
	var errMsgs []string
	
	// Create a mining reward transaction for the validator
	rewardAmount := bc.GetRewardAmount()
	if rewardAmount.Cmp(big.NewInt(0)) > 0 {
		// Convert big.Int to uint64 for the transaction
		rewardUint64 := uint64(0)
		if rewardAmount.IsUint64() {
			rewardUint64 = rewardAmount.Uint64()
		} else {
			// If reward is too large for uint64, cap it
			log.Printf("Warning: Reward amount is too large for uint64, capping it")
			rewardUint64 = ^uint64(0) // Maximum uint64 value
		}
		
		rewardTx := &Transaction{
			ID:        fmt.Sprintf("reward_%d_%s", block.Index, block.Validator),
			From:      "confirmix_genesis_address", // Rewards come from the genesis account
			To:        block.Validator,
			Value:     rewardUint64,
			Timestamp: block.Timestamp,
			Type:      "reward",
			Status:    "confirmed",
			BlockIndex: int64(block.Index),
			BlockHash:  block.Hash,
		}
		
		// Add the reward transaction to the block
		block.Transactions = append(block.Transactions, rewardTx)
		
		// Update balances for the reward transaction
		if err := bc.UpdateBalances(rewardTx); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to process reward transaction: %v", err))
		}
	}
	
	// Process all user transactions
	for _, tx := range block.Transactions {
		// Skip the reward transaction as it was already processed
		if tx.Type == "reward" {
			continue
		}
		
		// Update transaction status
		tx.Status = "confirmed"
		tx.BlockIndex = int64(block.Index)
		tx.BlockHash = block.Hash
		
		// Update balances
		if err := bc.UpdateBalances(tx); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to process transaction %s: %v", tx.ID, err))
			continue
		}
		
		// Process contract transaction if applicable
		if tx.IsContractTransaction() {
			if err := bc.processContractTransaction(tx); err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("failed to process contract transaction %s: %v", tx.ID, err))
			}
		}
	}
	
	// Clean transaction pool
	bc.cleanTransactionPool(block.Transactions)
	
	// Save blockchain state
	if err := bc.SaveToDisk(); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("failed to save blockchain state: %v", err))
	}
	
	if len(errMsgs) > 0 {
		return fmt.Errorf("block added with errors: %s", strings.Join(errMsgs, "; "))
	}
	
	return nil
}

// verifyBlockSignature verifies the signature of a block
func (bc *Blockchain) verifyBlockSignature(block *Block) error {
	// Get the public key for the block validator
	keyPair, exists := bc.keyPairs[block.Validator]
	if !exists {
		return errors.New("validator's public key not found")
	}
	
	return block.Verify(keyPair.PublicKey)
}

// processContractTransaction processes a contract transaction
func (bc *Blockchain) processContractTransaction(tx *Transaction) error {
	// Parse contract transaction data
	contractTx, err := ParseContractTransaction(tx.Data)
	if err != nil {
		return err
	}
	
	// Process based on contract operation
	switch contractTx.Operation {
	case "deploy":
		// Deploy a new contract
		_, err := bc.contractManager.DeployContract(contractTx.Code, tx.From)
		return err
		
	case "call":
		// Call a contract function
		_, err := bc.contractManager.CallContract(
			contractTx.ContractAddress,
			contractTx.Function,
			contractTx.Parameters,
			tx.From,
		)
		return err
		
	default:
		return errors.New("unknown contract operation")
	}
}

// cleanTransactionPool removes transactions that were included in a block
func (bc *Blockchain) cleanTransactionPool(txs []*Transaction) {
	for _, tx := range txs {
		delete(bc.txPool, tx.ID)
		
		// Also remove from pending transactions
		for i, pendingTx := range bc.pendingTxs {
			if pendingTx.ID == tx.ID {
				bc.pendingTxs = append(bc.pendingTxs[:i], bc.pendingTxs[i+1:]...)
				break
			}
		}
	}
}

// GetChainHeight returns the current height (length) of the blockchain
func (bc *Blockchain) GetChainHeight() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return uint64(len(bc.Blocks) - 1)
}

// GetLatestBlock returns the latest block in the blockchain
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlock returns a block by its hash
func (bc *Blockchain) GetBlock(hash string) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for _, block := range bc.Blocks {
		if block.Hash == hash {
			return block, nil
		}
	}
	
	return nil, errors.New("block not found")
}

// GetBlockByIndex returns a block by its index
func (bc *Blockchain) GetBlockByIndex(index uint64) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	if index >= uint64(len(bc.Blocks)) {
		return nil, errors.New("block index out of range")
	}
	
	return bc.Blocks[index], nil
}

// GetContractManager returns the contract manager
func (bc *Blockchain) GetContractManager() *ContractManager {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.contractManager
}

// GetTransaction returns a transaction by ID
func (bc *Blockchain) GetTransaction(id string) (*Transaction, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	tx, exists := bc.txPool[id]
	return tx, exists
}

// GetKeyPair returns the key pair for an address
func (bc *Blockchain) GetKeyPair(address string) (*KeyPair, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	keyPair, exists := bc.keyPairs[address]
	return keyPair, exists
}

// AddKeyPair adds a key pair for an address to the blockchain
func (bc *Blockchain) AddKeyPair(address string, keyPair *KeyPair) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.keyPairs[address] = keyPair
	
	// Save the blockchain state after adding a key pair
	go bc.SaveToDisk()
}

// GetAllAddresses returns all addresses with key pairs in the blockchain
func (bc *Blockchain) GetAllAddresses() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	addresses := make([]string, 0, len(bc.keyPairs))
	for addr := range bc.keyPairs {
		addresses = append(addresses, addr)
	}
	return addresses
}

// CreateAccount creates a new account with initial balance
func (bc *Blockchain) CreateAccount(address string, initialBalance *big.Int) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if _, exists := bc.accounts[address]; exists {
		return errors.New("account already exists")
	}
	
	bc.accounts[address] = initialBalance
	return nil
}

// GetBalance gets the balance of an account
func (bc *Blockchain) GetBalance(address string) (*big.Int, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	balance, exists := bc.accounts[address]
	if !exists {
		return big.NewInt(0), nil // Return 0 if account doesn't exist
	}
	
	return balance, nil
}

// UpdateBalances updates account balances based on a transaction
func (bc *Blockchain) UpdateBalances(tx *Transaction) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Convert transaction value to big.Int
	txValue := new(big.Int).SetUint64(tx.Value)
	
	// Reward transaction handling
	if tx.Type == "reward" {
		// Add rewards to validator account
		currentBalance, exists := bc.accounts[tx.To]
		if !exists {
			currentBalance = big.NewInt(0)
		}
		bc.accounts[tx.To] = new(big.Int).Add(currentBalance, txValue)
		return nil
	}
	
	// Regular transaction handling
	if tx.From == tx.To {
		return errors.New("sender and recipient cannot be the same")
	}
	
	fromBalance, exists := bc.accounts[tx.From]
	if !exists {
		return errors.New("sender account does not exist")
	}
	
	// Check if sender has enough funds
	if fromBalance.Cmp(txValue) < 0 {
		return errors.New("insufficient funds")
	}
	
	// Update sender's balance
	bc.accounts[tx.From] = new(big.Int).Sub(fromBalance, txValue)
	
	// Update recipient's balance
	toBalance, exists := bc.accounts[tx.To]
	if !exists {
		toBalance = big.NewInt(0)
	}
	bc.accounts[tx.To] = new(big.Int).Add(toBalance, txValue)
	
	return nil
}

// ValidatorInfo represents information about a validator
type ValidatorInfo struct {
	Address    string `json:"address"`
	HumanProof string `json:"humanProof"`
}

// GetValidators returns the list of registered validators
func (bc *Blockchain) GetValidators() []ValidatorInfo {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	validators := make([]ValidatorInfo, 0, len(bc.validators))
	for addr := range bc.validators {
		validators = append(validators, ValidatorInfo{
			Address:    addr,
			HumanProof: bc.humanProofs[addr],
		})
	}
	return validators
}

// RemoveTransaction removes a transaction from the pool by ID
func (bc *Blockchain) RemoveTransaction(txID string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if transaction exists in the pool
	if _, exists := bc.txPool[txID]; !exists {
		return fmt.Errorf("transaction %s not found in pool", txID)
	}
	
	// Remove from transaction pool
	delete(bc.txPool, txID)
	
	// Also remove from pending transactions
	for i, tx := range bc.pendingTxs {
		if tx.ID == txID {
			// Remove by replacing with the last element and then truncating
			bc.pendingTxs[i] = bc.pendingTxs[len(bc.pendingTxs)-1]
			bc.pendingTxs = bc.pendingTxs[:len(bc.pendingTxs)-1]
			break
		}
	}
	
	return nil
}

// GetRewardAmount returns the amount of ConX tokens to be rewarded for mining a block
// This implements a halving schedule for rewards
func (bc *Blockchain) GetRewardAmount() *big.Int {
	// Base reward is 50 ConX
	baseReward := new(big.Int)
	baseReward.SetString("50000000000000000000", 10) // 50 tokens with 18 decimals
	
	// Get current blockchain height
	height := bc.GetChainHeight()
	
	// Calculate halving (every 210,000 blocks, like Bitcoin)
	// This will reduce rewards by half approximately every 4 years
	// if blocks are mined every 10 minutes
	halvings := height / 210000
	
	// Apply halving (maximum 64 halvings to prevent underflow)
	if halvings >= 64 {
		return big.NewInt(0) // Rewards end after 64 halvings
	}
	
	// Calculate reward with halving applied
	for i := uint64(0); i < halvings; i++ {
		baseReward.Div(baseReward, big.NewInt(2))
	}
	
	return baseReward
}

// MineBlock creates a new block with all pending transactions and adds it to the blockchain
func (bc *Blockchain) MineBlock(validatorAddress string) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if validator is registered
	if !bc.validators[validatorAddress] {
		// If using genesis address for system operations, allow it
		if validatorAddress != "confirmix_genesis_address" {
			return nil, fmt.Errorf("address %s is not a registered validator", validatorAddress)
		}
	}
	
	// Get validator's human proof
	humanProof := bc.humanProofs[validatorAddress]
	if humanProof == "" && validatorAddress != "confirmix_genesis_address" {
		return nil, fmt.Errorf("human proof not found for validator %s", validatorAddress)
	}
	
	// For genesis address special case
	if validatorAddress == "confirmix_genesis_address" {
		humanProof = "genesis"
	}
	
	// Get all pending transactions
	pendingTxs := make([]*Transaction, len(bc.pendingTxs))
	copy(pendingTxs, bc.pendingTxs)
	
	// Maximum transactions per block (can be adjusted based on your requirements)
	const MAX_TRANSACTIONS_PER_BLOCK = 1000
	
	// Limit the number of transactions to avoid overly large blocks
	if len(pendingTxs) > MAX_TRANSACTIONS_PER_BLOCK {
		log.Printf("Limiting block to %d transactions out of %d pending", MAX_TRANSACTIONS_PER_BLOCK, len(pendingTxs))
		pendingTxs = pendingTxs[:MAX_TRANSACTIONS_PER_BLOCK]
	}
	
	// Create new block
	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := &Block{
		Index:        uint64(len(bc.Blocks)),
		Timestamp:    time.Now().Unix(),
		Transactions: pendingTxs,
		PrevHash:     lastBlock.Hash,
		Validator:    validatorAddress,
		HumanProof:   humanProof,
	}
	
	// Calculate block hash
	newBlock.Hash = newBlock.CalculateHash()
	
	// Sign the block if the validator is not genesis
	if validatorAddress != "confirmix_genesis_address" {
		keyPair, exists := bc.keyPairs[validatorAddress]
		if !exists {
			return nil, fmt.Errorf("key pair not found for validator %s", validatorAddress)
		}
		
		if err := newBlock.Sign(keyPair.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to sign block: %v", err)
		}
	}
	
	// Add the block to the blockchain
	bc.Blocks = append(bc.Blocks, newBlock)
	
	// Update the list of pending transactions - remove ones included in this block
	remainingTxs := make([]*Transaction, 0)
	for _, tx := range bc.pendingTxs {
		included := false
		for _, includedTx := range pendingTxs {
			if tx.ID == includedTx.ID {
				included = true
				break
			}
		}
		
		if !included {
			remainingTxs = append(remainingTxs, tx)
		}
	}
	bc.pendingTxs = remainingTxs
	
	// Process transactions to update balances
	for _, tx := range pendingTxs {
		if err := bc.UpdateBalances(tx); err != nil {
			log.Printf("Error updating balances for transaction %s: %v", tx.ID, err)
		}
	}
	
	// Create mining reward transaction
	rewardAmount := bc.GetRewardAmount()
	if rewardAmount.Cmp(big.NewInt(0)) > 0 && validatorAddress != "confirmix_genesis_address" {
		rewardTx := &Transaction{
			ID:        fmt.Sprintf("reward_%d_%s", newBlock.Index, validatorAddress),
			From:      "confirmix_genesis_address",
			To:        validatorAddress,
			Value:     rewardAmount.String(),
			Timestamp: time.Now().Unix(),
			Signature: "system_reward",
			Type:      "reward",
			Status:    "confirmed",
		}
		
		// Add reward transaction (no need to sign system transactions)
		if err := bc.UpdateBalances(rewardTx); err != nil {
			log.Printf("Error processing mining reward: %v", err)
		}
	}
	
	// Save blockchain state
	go bc.SaveToDisk()
	
	log.Printf("New block mined and added to blockchain: %d with %d transactions", newBlock.Index, len(pendingTxs))
	return newBlock, nil
}
