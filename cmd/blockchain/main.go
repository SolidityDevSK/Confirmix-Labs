package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/user/poa-poh-hybrid/pkg/blockchain"
	"github.com/user/poa-poh-hybrid/pkg/consensus"
	"github.com/user/poa-poh-hybrid/pkg/network"
)

// NodeConfig represents the node configuration
type NodeConfig struct {
	Address       string   `json:"address"`
	Port          int      `json:"port"`
	PrivateKeyPEM string   `json:"private_key_pem"`
	IsValidator   bool     `json:"is_validator"`
	HumanProof    string   `json:"human_proof"`
	PeerAddresses []string `json:"peer_addresses"`
}

func main() {
	// Define command line flags
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	validatorFlag := nodeCmd.Bool("validator", false, "Run as a validator")
	addressFlag := nodeCmd.String("address", "127.0.0.1", "Node address")
	portFlag := nodeCmd.Int("port", 8000, "Node port")
	configFlag := nodeCmd.String("config", "", "Configuration file path")
	peersFlag := nodeCmd.String("peers", "", "Comma-separated list of peer addresses")
	pohVerifyFlag := nodeCmd.Bool("poh-verify", false, "Enable PoH verification")

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Expected 'node' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "node":
		nodeCmd.Parse(os.Args[2:])
	default:
		fmt.Println("Expected 'node' subcommand")
		os.Exit(1)
	}

	// Load or create configuration
	config := &NodeConfig{
		Address:       *addressFlag,
		Port:          *portFlag,
		IsValidator:   *validatorFlag,
		PeerAddresses: []string{},
	}

	if *configFlag != "" {
		// Load configuration from file
		configData, err := ioutil.ReadFile(*configFlag)
		if err == nil {
			err = json.Unmarshal(configData, config)
			if err != nil {
				log.Fatalf("Failed to parse config file: %v", err)
			}
		}
	}

	// Parse peer addresses
	if *peersFlag != "" {
		// TODO: Parse comma-separated peer addresses
	}

	// Create or load private key
	privateKey, err := loadOrCreatePrivateKey(config)
	if err != nil {
		log.Fatalf("Failed to load or create private key: %v", err)
	}

	// Create node address from public key
	publicKey := privateKey.PublicKey
	publicKeyBytes := elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y)
	nodeAddress := fmt.Sprintf("%x", publicKeyBytes[:10]) // Use first 10 bytes of public key as address

	// Create blockchain
	bc := blockchain.NewBlockchain()

	// Create consensus engine
	hybridConsensus := consensus.NewHybridConsensus(bc, privateKey, nodeAddress, 15*time.Second)

	// Create P2P network node
	p2pNode := network.NewP2PNode(config.Address, config.Port, bc)

	// Initialize node
	initializeNode(config, hybridConsensus, p2pNode, *pohVerifyFlag)

	// Start P2P node
	err = p2pNode.Start()
	if err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	defer p2pNode.Stop()

	// Save configuration
	saveConfig(config)

	// Handle node startup based on configuration
	if config.IsValidator {
		err = hybridConsensus.StartMining()
		if err != nil {
			log.Printf("Failed to start mining: %v", err)
			if *pohVerifyFlag {
				// Initiate human verification if needed
				initiateHumanVerification(hybridConsensus)
			}
		}
	}

	// Connect to known peers
	for _, peerAddr := range config.PeerAddresses {
		err = p2pNode.ConnectToPeer(peerAddr)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", peerAddr, err)
		}
	}

	// Wait for interrupt signal
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	<-interruptChan

	// Cleanup
	hybridConsensus.StopMining()
	fmt.Println("Blockchain node stopped")
}

// loadOrCreatePrivateKey loads an existing private key or creates a new one
func loadOrCreatePrivateKey(config *NodeConfig) (*ecdsa.PrivateKey, error) {
	if config.PrivateKeyPEM != "" {
		// Load existing private key
		block, _ := pem.Decode([]byte(config.PrivateKeyPEM))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block containing private key")
		}

		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	}

	// Create new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	config.PrivateKeyPEM = string(privateKeyPEM)
	return privateKey, nil
}

// saveConfig saves the node configuration to a file
func saveConfig(config *NodeConfig) {
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}

	configDir := filepath.Join(os.Getenv("HOME"), ".poa-poh-hybrid")
	os.MkdirAll(configDir, 0755)

	configFile := filepath.Join(configDir, "config.json")
	err = ioutil.WriteFile(configFile, configData, 0644)
	if err != nil {
		log.Printf("Failed to save config: %v", err)
	}
}

// initializeNode initializes the node based on configuration
func initializeNode(config *NodeConfig, hybridConsensus *consensus.HybridConsensus, p2pNode *network.P2PNode, pohVerify bool) {
	if config.IsValidator {
		if config.HumanProof != "" && pohVerify {
			// Attempt to complete human verification with existing proof
			err := hybridConsensus.CompleteHumanVerification(config.HumanProof)
			if err != nil {
				log.Printf("Failed to verify with existing human proof: %v", err)
				initiateHumanVerification(hybridConsensus)
			}
		} else if pohVerify {
			initiateHumanVerification(hybridConsensus)
		}
	}
}

// initiateHumanVerification initiates the human verification process
func initiateHumanVerification(hybridConsensus *consensus.HybridConsensus) {
	proofToken, err := hybridConsensus.InitiateHumanVerification()
	if err != nil {
		log.Printf("Failed to initiate human verification: %v", err)
		return
	}

	fmt.Println("Human verification required to become a validator")
	fmt.Printf("Your proof token is: %s\n", proofToken)
	fmt.Println("Please complete the verification process at [verification URL]")
	fmt.Println("After verification, restart the node with --config flag")
} 