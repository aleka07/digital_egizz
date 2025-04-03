// cmd/apiserver/main.go
package main

import (
	"context" // Need context for DB connection
	"log"
	"net/http"
	"os"        // For environment variables
	"os/signal" // For graceful shutdown
	"syscall"   // For system signals
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/api"         // Import our api package
	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/persistence" // Import our persistence package
)

func main() {
	log.Println("INFO: Starting Digital Twin Framework API Server...")

	// --- Configuration ---
	// Get Database DSN from environment variable
	dbDSN := os.Getenv("DATABASE_DSN")
	if dbDSN == "" {
		// Provide a sensible default for local development (adjust as needed)
		dbDSN = "postgres://user:password@localhost:5432/digital_twin_db?sslmode=disable"
		log.Printf("WARN: DATABASE_DSN environment variable not set. Using default: %s", dbDSN)
		// Consider logging dbDSN without password in production logs
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	// --- Create Dependencies ---
	// Context for initialization tasks
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10-sec timeout for DB connection
	defer cancel()

	// Create the persistence layer (Postgres store)
	modelStore, err := persistence.NewPostgresModelStore(initCtx, dbDSN)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database connection: %v", err)
	}
	// Defer closing the store until main() exits
	defer modelStore.Close()

	// Create the API handler, injecting the *DB-backed* store
	// Note: api.API now needs adjustment to accept the persistence.ModelStore interface
	apiHandler := api.NewAPI(modelStore) // <<< We need to adjust api.NewAPI

	// --- Create Router (using chi) ---
	r := chi.NewRouter()

	// --- Middleware ---
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Consider a more structured logger later
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// --- Register Routes (No change needed here if API handlers use the interface correctly)---
	r.Get("/healthz", api.HealthCheckHandler) // TODO: Enhance health check to ping DB?

	r.Route("/api/v1/models", func(r chi.Router) {
		// These handlers now operate on the PostgresModelStore via the interface
		r.Get("/", apiHandler.ListModels)
		r.Post("/", apiHandler.CreateModel)
		r.Get("/{modelId}", apiHandler.GetModel)
		r.Put("/{modelId}", apiHandler.UpdateModel)
		r.Delete("/{modelId}", apiHandler.DeleteModel)
	})

	// --- Configure and Start Server ---
	server := &http.Server{
		Addr:         ":" + apiPort, // Use configured port
		Handler:      r,
		ReadTimeout:  15 * time.Second, // Slightly increased timeouts
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for server errors
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Printf("INFO: Server listening on :%s", apiPort)
		serverErrors <- server.ListenAndServe()
	}()

	// --- Graceful Shutdown ---
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM) // Listen for Ctrl+C or kill

	// Block until either a server error or a shutdown signal is received
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			log.Fatalf("FATAL: Server error: %v", err)
		} else {
			log.Println("INFO: Server stopped via ServerError (likely shutdown).")
		}
	case sig := <-shutdown:
		log.Printf("INFO: Shutdown signal (%v) received. Starting graceful shutdown...", sig)

		// Create a context with timeout for shutdown
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second) // Allow 15s for shutdown
		defer cancelShutdown()

		// Attempt to gracefully shut down the server
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("ERROR: Graceful server shutdown failed: %v", err)
			// Force close if shutdown fails
			if closeErr := server.Close(); closeErr != nil {
				log.Printf("ERROR: Server Close() failed: %v", closeErr)
			}
		} else {
			log.Println("INFO: Server shutdown complete.")
		}
	}

	// modelStore.Close() is called here via defer
	log.Println("INFO: Application shutdown finished.")
}
