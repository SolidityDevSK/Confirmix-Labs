package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	accounts       map[string]float64 // Map of account addresses to balances
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
		accounts:       make(map[string]float64),
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
	
	// Save accounts
	accountsFile := filepath.Join(dataDir, "accounts.json")
	accountsData, err := json.MarshalIndent(bc.accounts, "", "  ")
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
	if _, err := os.Stat(accountsFile); !os.IsNotExist(err) {
		accountsData, err := ioutil.ReadFile(accountsFile)
		if err == nil {
			var accounts map[string]float64
			if err := json.Unmarshal(accountsData, &accounts); err == nil {
				bc.accounts = accounts
			}
		}
	}
	
	log.Printf("Blockchain state loaded from disk: %d blocks", len(bc.Blocks))
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
	// Temel doğrulama kontrolleri (mutex almadan önce yapabiliriz)
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}
	
	if tx.From == "" {
		return errors.New("sender address cannot be empty")
	}
	
	if tx.To == "" {
		return errors.New("recipient address cannot be empty")
	}
	
	if tx.Value <= 0 {
		return fmt.Errorf("invalid transaction amount: %f", tx.Value)
	}
	
	// Mutex'i kısa sürede al ve bırak
	bc.mu.Lock()
	
	// Check if transaction already exists
	if _, exists := bc.txPool[tx.ID]; exists {
		bc.mu.Unlock()
		return errors.New("transaction already exists in the pool")
	}
	
	// Alıcı hesap var mı kontrol et - kilidi bırakmadan önce bilgiyi al
	_, aliciHesapVar := bc.accounts[tx.To]
	
	// Kilidi bırak
	bc.mu.Unlock()
	
	// Tekrar kilidi al
	bc.mu.Lock()
	
	// Alıcı hesap yoksa oluştur
	if !aliciHesapVar {
		bc.accounts[tx.To] = 0
	}
	
	// Transaction'ın bir kopyasını oluşturup, txPool ve pendingTxs'e ekle
	txCopy := *tx
	bc.txPool[tx.ID] = &txCopy
	bc.pendingTxs = append(bc.pendingTxs, &txCopy)
	
	// İşlem tamamlandı, mutex'i bırak
	bc.mu.Unlock()
	
	// Save blockchain state to disk after adding a new transaction
	go bc.SaveToDisk()
	
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
	
	// Basic validation
	if len(bc.Blocks) > 0 {
		lastBlock := bc.Blocks[len(bc.Blocks)-1]
		
		// Check if block links to the previous block
		if block.PrevHash != lastBlock.Hash {
			return errors.New("invalid previous hash")
		}
		
		// Check if block index is sequential
		if block.Index != lastBlock.Index+1 {
			return errors.New("invalid block index")
		}
		
		// Check if validator is authorized
		if !bc.validators[block.Validator] {
			return errors.New("unauthorized validator")
		}
		
		// Check if human proof matches the validator's registered proof
		// Human proof kontrolü geçici olarak devre dışı bırakıldı
		// if block.HumanProof != bc.humanProofs[block.Validator] {
		//     return errors.New("invalid human proof")
		// }

		// Verify block hash
		if block.Hash != block.CalculateHash() {
			return errors.New("invalid block hash")
		}

		// Verify block signature
		if err := bc.verifyBlockSignature(block); err != nil {
			return err
		}
	}
	
	// Process transactions in the block (execute smart contracts)
	for _, tx := range block.Transactions {
		if tx.Data != nil && len(tx.Data) > 0 {
			// Check if this is a contract transaction
			if tx.IsContractTransaction() {
				err := bc.processContractTransaction(tx)
				if err != nil {
					return err
				}
			}
		}
	}
	
	// Add block to blockchain
	bc.Blocks = append(bc.Blocks, block)
	
	// Remove transactions that were included in the block
	bc.cleanTransactionPool(block.Transactions)
	
	// Save blockchain state to disk after adding a new block
	go bc.SaveToDisk()
	
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

// CreateAccount creates a new account with an initial balance
func (bc *Blockchain) CreateAccount(address string, initialBalance float64) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if _, exists := bc.accounts[address]; exists {
		return errors.New("account already exists")
	}
	bc.accounts[address] = initialBalance
	
	// Save the blockchain state after creating an account
	go bc.SaveToDisk()
	
	return nil
}

// GetBalance returns the balance of the given account
func (bc *Blockchain) GetBalance(address string) (float64, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	balance, exists := bc.accounts[address]
	if !exists {
		return 0, errors.New("account does not exist")
	}
	
	return balance, nil
}

// UpdateBalances updates the balances of the sender and receiver after a transaction
func (bc *Blockchain) UpdateBalances(tx *Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Hesap kontrolü - göndericinin hesabı var mı?
	senderBalance, senderExists := bc.accounts[tx.From]
	if !senderExists {
		return fmt.Errorf("sender account %s does not exist", tx.From)
	}
	
	// Bakiye kontrolü
	if tx.Value <= 0 {
		return fmt.Errorf("invalid transaction value: %f", tx.Value)
	}
	
	if tx.Value > senderBalance {
		return fmt.Errorf("insufficient balance: required %f, available %f", tx.Value, senderBalance)
	}
	
	// Alıcı hesabı kontrol et, yoksa oluştur
	_, receiverExists := bc.accounts[tx.To]
	if !receiverExists {
		bc.accounts[tx.To] = 0
	}
	
	// Bakiyeleri güncelle
	bc.accounts[tx.From] -= tx.Value
	bc.accounts[tx.To] += tx.Value
	
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
