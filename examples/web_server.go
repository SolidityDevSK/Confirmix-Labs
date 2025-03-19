package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"time"

	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/api"
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/blockchain"
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/consensus"
	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/util"
)

func main() {
	// Generate a private key for the node
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	
	// Generate address from public key
	address := util.PublicKeyToAddress(&privateKey.PublicKey)
	
	// Create a new blockchain
	bc := blockchain.NewBlockchain()

	// Create a new hybrid consensus engine with 5-second block time
	blockTime := 5 * time.Second
	ce := consensus.NewHybridConsensus(bc, privateKey, address, blockTime)

	// Register as validator with human verification
	proofToken, err := ce.InitiateHumanVerification()
	if err != nil {
		log.Fatalf("Failed to initiate human verification: %v", err)
	}

	// Complete verification
	if err := ce.CompleteHumanVerification(proofToken); err != nil {
		log.Fatalf("Failed to complete human verification: %v", err)
	}

	// Register as validator
	if err := ce.RegisterAsValidator(); err != nil {
		log.Fatalf("Failed to register as validator: %v", err)
	}

	// Add node to validator list
	ce.UpdateValidatorList([]string{address})

	// Start the consensus engine
	go ce.StartMining()

	// Create a new web server
	ws := api.NewWebServer(bc, ce, 8080)

	// Start the web server
	log.Println("Starting web server on port 8080...")
	if err := ws.Start(); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
} 