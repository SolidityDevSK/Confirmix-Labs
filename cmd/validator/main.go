package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ConfirmixLabs/Confirmix-Labs/pkg/validator"
)

type ValidatorInfo struct {
	Address    string `json:"address"`
	HumanProof string `json:"humanProof"`
}

func main() {
	// Command line flags
	address := flag.String("address", "", "Validator address (wallet address to use for validation)")
	apiURL := flag.String("api", "http://localhost:8080/api", "API base URL of the blockchain node")
	interval := flag.Int("interval", 10, "Validation interval in seconds")
	flag.Parse()

	// Validate inputs
	if *address == "" {
		fmt.Println("Error: Validator address is required")
		fmt.Println("Usage: validator -address=<validator_address> [-api=<api_url>] [-interval=<seconds>]")
		os.Exit(1)
	}

	// Verify the address is a validator
	isValidator, err := checkValidatorStatus(*apiURL, *address)
	if err != nil {
		log.Printf("Warning: Could not verify validator status: %v", err)
	} else if !isValidator {
		log.Printf("Error: Address %s is not registered as a validator", *address)
		log.Printf("Please register as a validator first through the web interface")
		os.Exit(1)
	}

	// Create validator
	validatorNode := validator.NewValidator(*apiURL, *address, time.Duration(*interval)*time.Second)

	// Start validator
	if err := validatorNode.Start(); err != nil {
		log.Fatalf("Failed to start validator: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Print startup message
	fmt.Printf("Validator started with address: %s\n", *address)
	fmt.Printf("Connected to API: %s\n", *apiURL)
	fmt.Printf("Validation interval: %d seconds\n", *interval)
	fmt.Println("Press Ctrl+C to exit")

	// Wait for termination signal
	<-sigChan
	fmt.Println("\nShutting down validator...")
	validatorNode.Stop()
	fmt.Println("Validator stopped")
}

// checkValidatorStatus checks if the address is a registered validator
func checkValidatorStatus(apiURL, address string) (bool, error) {
	// Fix API URL if needed
	apiURL = strings.TrimSuffix(apiURL, "/")
	
	// Get validators list
	validatorsURL := fmt.Sprintf("%s/validators", apiURL)
	log.Printf("Requesting validators from: %s", validatorsURL)
	
	resp, err := http.Get(validatorsURL)
	if err != nil {
		return false, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}
	
	var validators []ValidatorInfo
	if err := json.NewDecoder(resp.Body).Decode(&validators); err != nil {
		return false, fmt.Errorf("failed to decode validators: %w", err)
	}
	
	// Check if address is in validators list
	for _, v := range validators {
		if v.Address == address {
			return true, nil
		}
	}
	
	return false, nil
} 