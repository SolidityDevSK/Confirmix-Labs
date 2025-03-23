package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
)

// KeyPair represents a public-private key pair
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// NewKeyPair generates a new ECDSA key pair
func NewKeyPair() (*KeyPair, error) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// SignTransaction signs a transaction with the given private key
func (tx *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
	// Create a hash of the transaction data
	hash := tx.CalculateHash()
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, []byte(hash))
	if err != nil {
		return err
	}

	// Combine r and s into a single signature
	signature := append(r.Bytes(), s.Bytes()...)
	tx.Signature = signature
	return nil
}

// VerifyTransaction verifies the transaction signature
func (tx *Transaction) Verify(publicKey *ecdsa.PublicKey) error {
	if tx.Signature == nil || len(tx.Signature) == 0 {
		return errors.New("transaction is not signed")
	}

	// Split signature into r and s components
	r := new(big.Int).SetBytes(tx.Signature[:len(tx.Signature)/2])
	s := new(big.Int).SetBytes(tx.Signature[len(tx.Signature)/2:])

	// Create a hash of the transaction data
	hash := tx.CalculateHash()

	// Verify the signature
	valid := ecdsa.Verify(publicKey, []byte(hash), r, s)
	if !valid {
		return errors.New("invalid transaction signature")
	}

	return nil
}

// CalculateHash calculates the hash of a transaction for signing
func (tx *Transaction) CalculateHash() string {
	// Create a string representation of the transaction
	data := tx.ID + tx.From + tx.To + string(IntToHex(int64(tx.Value * 1000000)))
	if tx.Data != nil {
		data += string(tx.Data)
	}
	data += string(IntToHex(tx.Timestamp))

	// Calculate SHA-256 hash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetPublicKeyFromAddress converts an address to a public key
func GetPublicKeyFromAddress(address string) (*ecdsa.PublicKey, error) {
	// In a real implementation, this would decode the address and reconstruct the public key
	// For now, we'll return an error as this is a placeholder
	return nil, errors.New("address to public key conversion not implemented")
}

// GetPublicKeyString returns the public key as a hexadecimal string
func (kp *KeyPair) GetPublicKeyString() string {
	if kp.PublicKey == nil {
		return ""
	}
	publicKeyBytes := elliptic.Marshal(kp.PublicKey.Curve, kp.PublicKey.X, kp.PublicKey.Y)
	return hex.EncodeToString(publicKeyBytes)
}

// GetPrivateKeyString returns the private key as a hexadecimal string
func (kp *KeyPair) GetPrivateKeyString() string {
	if kp.PrivateKey == nil {
		return ""
	}
	return hex.EncodeToString(kp.PrivateKey.D.Bytes())
} 