package blockchain

import (
	"errors"
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
	}

	// Create the genesis block
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
	return bc
}

// AddValidator adds a new authorized validator to the blockchain
func (bc *Blockchain) AddValidator(address string, humanProof string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if human proof is valid (in a real implementation, this would verify with PoH)
	if humanProof == "" {
		return errors.New("human proof is required for validators")
	}
	
	bc.validators[address] = true
	bc.humanProofs[address] = humanProof
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
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if transaction already exists
	if _, exists := bc.txPool[tx.ID]; exists {
		return errors.New("transaction already exists")
	}
	
	bc.txPool[tx.ID] = tx
	bc.pendingTxs = append(bc.pendingTxs, tx)
	return nil
}

// GetPendingTransactions gets pending transactions ready to be included in a block
func (bc *Blockchain) GetPendingTransactions(limit int) []*Transaction {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	if limit > len(bc.pendingTxs) {
		limit = len(bc.pendingTxs)
	}
	
	return bc.pendingTxs[:limit]
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
		if block.HumanProof != bc.humanProofs[block.Validator] {
			return errors.New("invalid human proof")
		}

		// Verify block hash
		if block.Hash != block.CalculateHash() {
			return errors.New("invalid block hash")
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
	
	return nil
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
	return bc.contractManager
}
