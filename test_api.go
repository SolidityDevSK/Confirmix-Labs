package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	// Our API endpoints
	baseURL := "http://localhost:8080/api"
	endpoints := map[string]string{
		"status":       baseURL + "/status",
		"blocks":       baseURL + "/blocks",
		"transactions": baseURL + "/transactions",
	}

	// Test each endpoint
	for name, url := range endpoints {
		fmt.Printf("Testing %s endpoint: %s\n", name, url)
		
		// Send GET request
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		// Read response body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}
		
		// Pretty-print JSON response
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, body, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting JSON: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
		} else {
			fmt.Printf("Response:\n%s\n\n", prettyJSON.String())
		}
	}

	// Test creating a new transaction
	fmt.Println("Testing transaction creation...")
	
	// Transaction data
	txData := map[string]interface{}{
		"from":  "test-sender",
		"to":    "test-receiver",
		"value": 25.0,
	}
	
	// Convert to JSON
	txJSON, err := json.Marshal(txData)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	
	// Send POST request
	resp, err := http.Post(
		baseURL+"/transaction", 
		"application/json", 
		bytes.NewBuffer(txJSON),
	)
	if err != nil {
		fmt.Printf("Error creating transaction: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	
	// Pretty-print JSON response
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
	} else {
		fmt.Printf("Transaction created:\n%s\n\n", prettyJSON.String())
	}
	
	// Check pending transactions after creating a transaction
	fmt.Println("Checking pending transactions after creation...")
	resp, err = http.Get(baseURL + "/transactions")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	
	// Pretty-print JSON response
	prettyJSON.Reset()
	err = json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting JSON: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
	} else {
		fmt.Printf("Pending Transactions:\n%s\n", prettyJSON.String())
	}
} 