package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"math/big"
	"time"
)

// Block represents a single block in the blockchain
type Block struct {
	Index        uint64
	Timestamp    int64
	Transactions []*Transaction
	Hash         string
	PrevHash     string
	Validator    string
	Signature    []byte
	Nonce        uint64
	HumanProof   string // Proof of Humanity verification marker
}

// CalculateHash calculates the hash of the block
func (b *Block) CalculateHash() string {
	record := bytes.Join(
		[][]byte{
			[]byte(b.PrevHash),
			[]byte(b.Validator),
			SerializeTransactions(b.Transactions),
			IntToHex(b.Timestamp),
			IntToHex(int64(b.Nonce)),
			[]byte(b.HumanProof),
		},
		[]byte{},
	)

	h := sha256.New()
	h.Write(record)
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// Transaction represents a transfer of data or value
// type Transaction struct {
// 	ID        string
// 	From      string
// 	To        string
// 	Value     float64
// 	Data      []byte
// 	Timestamp int64
// 	Signature []byte
// 	Type      string // "regular", "contract_deploy", "contract_call"
// }

// NewTransaction creates a new transaction
// func NewTransaction(from string, to string, value float64, data []byte) *Transaction {
// 	txType := "regular"
// 	if data != nil && len(data) > 0 {
// 		// Try to parse as contract transaction to determine type
// 		contractTx, err := ParseContractTransaction(data)
// 		if err == nil {
// 			if contractTx.Operation == "deploy" {
// 				txType = "contract_deploy"
// 			} else if contractTx.Operation == "call" {
// 				txType = "contract_call"
// 			}
// 		}
// 	}

// 	tx := &Transaction{
// 		From:      from,
// 		To:        to,
// 		Value:     value,
// 		Data:      data,
// 		Timestamp: time.Now().Unix(),
// 		Type:      txType,
// 	}

// 	// Generate ID based on transaction content
// 	h := sha256.New()
// 	h.Write([]byte(from))
// 	h.Write([]byte(to))
// 	h.Write(IntToHex(int64(value * 1000000))) // Convert float to int for consistent hashing
// 	if data != nil {
// 		h.Write(data)
// 	}
// 	h.Write(IntToHex(tx.Timestamp))
// 	tx.ID = hex.EncodeToString(h.Sum(nil))

// 	return tx
// }

// IsContractTransaction checks if this is a smart contract related transaction
// func (tx *Transaction) IsContractTransaction() bool {
// 	return tx.Type == "contract_deploy" || tx.Type == "contract_call"
// }

// ContractTransaction represents a transaction related to smart contracts
// type ContractTransaction struct {
// 	Operation       string        `json:"operation"` // "deploy" or "call"
// 	ContractAddress string        `json:"contract_address,omitempty"`
// 	Function        string        `json:"function,omitempty"`
// 	Parameters      []interface{} `json:"parameters,omitempty"`
// 	Code            string        `json:"code,omitempty"`
// }

// NewContractDeploymentTransaction creates a transaction to deploy a new contract
// func NewContractDeploymentTransaction(from string, code string) (*Transaction, error) {
// 	contractTx := ContractTransaction{
// 		Operation: "deploy",
// 		Code:      code,
// 	}

// 	data, err := json.Marshal(contractTx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return NewTransaction(from, "", 0, data), nil
// }

// NewContractCallTransaction creates a transaction to call a contract function
// func NewContractCallTransaction(from string, contractAddress string, function string, params []interface{}) (*Transaction, error) {
// 	contractTx := ContractTransaction{
// 		Operation:       "call",
// 		ContractAddress: contractAddress,
// 		Function:        function,
// 		Parameters:      params,
// 	}

// 	data, err := json.Marshal(contractTx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return NewTransaction(from, contractAddress, 0, data), nil
// }

// ParseContractTransaction parses contract transaction data
// func ParseContractTransaction(data []byte) (*ContractTransaction, error) {
// 	if data == nil || len(data) == 0 {
// 		return nil, errors.New("empty transaction data")
// 	}

// 	var contractTx ContractTransaction
// 	err := json.Unmarshal(data, &contractTx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &contractTx, nil
// }

// SerializeTransactions converts transactions to bytes for hashing
func SerializeTransactions(txs []*Transaction) []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	// Create a simplified structure for serialization
	type SimpleTx struct {
		ID    string
		From  string
		To    string
		Value uint64
		Data  []byte
		Type  string
	}

	simpleTxs := make([]SimpleTx, len(txs))
	for i, tx := range txs {
		simpleTxs[i] = SimpleTx{
			ID:    tx.ID,
			From:  tx.From,
			To:    tx.To,
			Value: tx.Value,
			Data:  tx.Data,
			Type:  tx.Type,
		}
	}

	err := encoder.Encode(simpleTxs)
	if err != nil {
		return []byte{}
	}
	return result.Bytes()
}

// NewBlock creates a new block in the blockchain
func NewBlock(index uint64, transactions []*Transaction, prevHash string, validator string, humanProof string) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Validator:    validator,
		HumanProof:   humanProof,
		Nonce:        0,
	}
	block.Hash = block.CalculateHash()
	return block
}

// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	return []byte(hex.EncodeToString([]byte{
		byte(num >> 56),
		byte(num >> 48),
		byte(num >> 40),
		byte(num >> 32),
		byte(num >> 24),
		byte(num >> 16),
		byte(num >> 8),
		byte(num),
	}))
}

// Sign signs the block with the given private key
func (b *Block) Sign(privateKey *ecdsa.PrivateKey) error {
	// Create a hash of the block data
	hash := b.CalculateHash()
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, []byte(hash))
	if err != nil {
		return err
	}

	// Combine r and s into a single signature
	signature := append(r.Bytes(), s.Bytes()...)
	b.Signature = signature
	return nil
}

// Verify verifies the block signature
func (b *Block) Verify(publicKey *ecdsa.PublicKey) error {
	if b.Signature == nil || len(b.Signature) == 0 {
		return errors.New("block is not signed")
	}

	// Split signature into r and s components
	r := new(big.Int).SetBytes(b.Signature[:len(b.Signature)/2])
	s := new(big.Int).SetBytes(b.Signature[len(b.Signature)/2:])

	// Create a hash of the block data
	hash := b.CalculateHash()

	// Verify the signature
	valid := ecdsa.Verify(publicKey, []byte(hash), r, s)
	if !valid {
		return errors.New("invalid block signature")
	}

	return nil
} 