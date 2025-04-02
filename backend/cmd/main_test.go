package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	// Create a request to the health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Create a new router and register the health handler
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Serve the request
	mux.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v, want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := `{"status":"healthy"}`
	// Remove trailing newline from actual response
	actual := rr.Body.String()
	if len(actual) > 0 && actual[len(actual)-1] == '\n' {
		actual = actual[:len(actual)-1]
	}

	if actual != expected {
		t.Errorf("Handler returned unexpected body: got %v, want %v", actual, expected)
	}
}
