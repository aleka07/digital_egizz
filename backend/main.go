package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	// Simple health check endpoint
	fmt.Fprintf(w, "OK")
}

func main() {
	listenAddr := ":8081"
	log.Printf("Backend server starting on %s", listenAddr)

	http.HandleFunc("/health", healthCheck)

	// Start the server
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}
