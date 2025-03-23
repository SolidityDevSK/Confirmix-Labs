package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func main() {
	// Echo sunucusunun URL'leri
	baseURL := "http://localhost:8081"
	endpoints := map[string]string{
		"echo":         baseURL + "/echo",
		"transaction":  baseURL + "/api/transaction",
		"transactions": baseURL + "/api/transactions",
	}
	
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	
	// Her endpoint'i test et
	for name, url := range endpoints {
		fmt.Printf("Testing %s endpoint: %s\n", name, url)
		
		var req *http.Request
		var err error
		
		if name == "transaction" {
			// POST request
			jsonData := `{"from":"test","to":"test","value":10}`
			req, err = http.NewRequest("POST", url, strings.NewReader(jsonData))
			req.Header.Add("Content-Type", "application/json")
		} else {
			// GET request
			req, err = http.NewRequest("GET", url, nil)
		}
		
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			continue
		}
		
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
			continue
		}
		
		// Response body'yi oku
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}
		
		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("Response body: %s\n\n", string(body))
	}
} 