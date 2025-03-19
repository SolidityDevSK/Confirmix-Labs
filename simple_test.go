package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func main() {
	// Basit bir işlem oluşturma testi
	url := "http://localhost:8080/api/transaction"
	
	// 2 saniye timeout ile bir HTTP istemcisi oluştur
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	
	jsonData := `{
		"from": "test-sender-999",
		"to": "test-receiver-999",
		"value": 9.99
	}`
	
	// Tek bir POST isteği gönder
	fmt.Printf("Sending POST request to %s\n", url)
	req, _ := http.NewRequest("POST", url, strings.NewReader(jsonData))
	req.Header.Add("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Yanıtı basitçe işle
	fmt.Printf("Response status: %s\n", resp.Status)
	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Transaction created successfully!")
	} else {
		fmt.Println("Failed to create transaction.")
	}
} 