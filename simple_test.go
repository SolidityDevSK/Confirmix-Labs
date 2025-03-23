package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"testing"
)

func main() {
	// Simple transaction creation test
	url := "http://localhost:8080/api/transaction"
	
	// Create an HTTP client with 2 second timeout
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	
	jsonData := `{
		"from": "test-sender-999",
		"to": "test-receiver-999",
		"value": 9.99
	}`
	
	// Send a single POST request
	fmt.Printf("Sending POST request to %s\n", url)
	req, _ := http.NewRequest("POST", url, strings.NewReader(jsonData))
	req.Header.Add("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Simply process the response
	fmt.Printf("Response status: %s\n", resp.Status)
	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Transaction created successfully!")
	} else {
		fmt.Println("Failed to create transaction.")
	}
}

// Simple transaction creation test
func TestCreateTransaction(t *testing.T) {
	// Create an HTTP client with 2 second timeout
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	
	jsonData := `{
		"from": "test-sender-999",
		"to": "test-receiver-999",
		"value": 9.99
	}`
	
	// Send a single POST request
	fmt.Printf("Sending POST request to %s\n", url)
	req, _ := http.NewRequest("POST", url, strings.NewReader(jsonData))
	req.Header.Add("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Simply process the response
	fmt.Printf("Response status: %s\n", resp.Status)
	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Transaction created successfully!")
	} else {
		fmt.Println("Failed to create transaction.")
	}
} 