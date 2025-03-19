package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/elliptic"
	"encoding/hex"
)

// Wallet represents a user's wallet with a key pair
type Wallet struct {
	Address    string
	KeyPair    *KeyPair
}

// CreateWallet creates a new wallet and returns the wallet address
func CreateWallet() (*Wallet, error) {
	keyPair, err := NewKeyPair()
	if err != nil {
		return nil, err
	}

	address := GenerateAddress(keyPair.PublicKey)
	wallet := &Wallet{
		Address: address,
		KeyPair: keyPair,
	}

	return wallet, nil
}

// GenerateAddress generates a blockchain address from a public key
func GenerateAddress(pubKey *ecdsa.PublicKey) string {
	hash := sha256.Sum256(elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y))
	return hex.EncodeToString(hash[:])
} 