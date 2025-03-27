package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"math/big"
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

// ImportPrivateKey reconstructs a private key from a hex string
func ImportPrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %v", err)
	}

	curve := elliptic.P256()
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyBytes)
	
	return privateKey, nil
}

// GenerateAddress generates a blockchain address from a public key
func GenerateAddress(pubKey *ecdsa.PublicKey) string {
	hash := sha256.Sum256(elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y))
	return hex.EncodeToString(hash[:])
} 