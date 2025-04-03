// cmd/apiserver/main.go
package main

import (
	"encoding/json" // For JSON response
	"log"           // For logging
	"net/http"      // For HTTP server
	"time"          // For timestamps
)

// healthCheckHandler responds to health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Set the content type header
	w.Header().Set("Content-Type", "application/json")

	// Prepare the response data
	response := map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Encode the response data as JSON and write it to the response writer
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If encoding fails, log the error and send an internal server error status
		log.Printf("ERROR: Failed to encode health check response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return // Important to return after writing an error
	}

	// Log the successful request (optional, can be verbose)
	// log.Printf("INFO: Served health check request from %s", r.RemoteAddr)
}

func main() {
	log.Println("INFO: Starting Digital Twin Framework API Server...")

	// --- Register Handlers ---
	// We create a ServeMux (router) to explicitly register handlers.
	// This is generally better practice than using the DefaultServeMux.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthCheckHandler) // Register handler for the /healthz path

	// --- TODO: Register other API endpoints here later ---
	// mux.HandleFunc("/api/v1/models", modelHandler)
	// mux.HandleFunc("/api/v1/twins", twinHandler)

	// --- Configure Server ---
	// Define the port (make this configurable later, e.g., via env vars or flags)
	port := ":8080"

	// Create the HTTP server configuration
	server := &http.Server{
		Addr:         port,
		Handler:      mux,              // Use our custom mux
		ReadTimeout:  10 * time.Second, // Add some basic timeouts
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("INFO: Server listening on %s", port)

	// --- Start Server ---
	// Start listening and serving requests
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		// Log a fatal error if the server fails to start (excluding graceful shutdown)
		log.Fatalf("FATAL: Could not start server: %v", err)
	}

	log.Println("INFO: Server stopped gracefully.")
}
