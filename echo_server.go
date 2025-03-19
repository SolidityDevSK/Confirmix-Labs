package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Echo handler
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you requested: %s\n", r.URL.Path)
	})

	// Test transaction endpoint
	http.HandleFunc("/api/transaction", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id":"test-tx-123","status":"created"}`)
	})

	// Test transactions endpoint
	http.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `[{"id":"test-tx-123","from":"sender","to":"receiver","value":10.0}]`)
	})

	// Start server on port 8081
	port := ":8081"
	fmt.Printf("Starting echo server on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
} 