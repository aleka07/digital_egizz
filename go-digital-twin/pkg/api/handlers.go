// pkg/api/handlers.go
package api

import (
	// Handlers now need context for store methods
	"encoding/json"
	"errors" // For checking specific persistence errors
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/model"
	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/persistence" // Import persistence package
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid" // Keep for ID generation if needed
)

// --- API Handler ---

// API holds dependencies. Now uses the ModelStore interface.
type API struct {
	ModelStore persistence.ModelStore // Use the interface type
	// Add other dependencies later (e.g., TwinStore, DB connections)
}

// NewAPI creates a new API handler structure, accepting the interface.
func NewAPI(modelStore persistence.ModelStore) *API { // Accept interface
	return &API{
		ModelStore: modelStore,
	}
}

// --- Model Handlers (Updated to use interface and context) ---

// CreateModel handles POST requests to /models
func (a *API) CreateModel(w http.ResponseWriter, r *http.Request) {
	var newModel model.TwinModel // Note: We are creating the struct here

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&newModel); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if newModel.ID == "" {
		newModel.ID = "model-" + uuid.NewString()
	}
	if newModel.DisplayName == "" {
		http.Error(w, "Missing required field: displayName", http.StatusBadRequest)
		return
	}

	// Set timestamps before storing
	now := time.Now().UTC()
	newModel.CreatedAt = now // Set creation time here
	newModel.UpdatedAt = now // Set initial update time here

	// --- Store the model using the interface ---
	// Use request context, potentially add timeout
	ctx := r.Context()                              // Get context from request
	err := a.ModelStore.CreateModel(ctx, &newModel) // Pass pointer
	if err != nil {
		log.Printf("ERROR: Failed to create model in store: %v", err)
		// Check for specific persistence errors
		if errors.Is(err, persistence.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, "Failed to create model", http.StatusInternalServerError)
		}
		return
	}
	// --- End Store ---

	log.Printf("INFO: Created model: ID=%s, Name=%s", newModel.ID, newModel.DisplayName)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newModel); err != nil {
		log.Printf("ERROR: Failed to encode create model response: %v", err)
	}
}

// GetModel handles GET requests to /models/{modelId}
func (a *API) GetModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "modelId")
	if modelID == "" {
		http.Error(w, "Missing modelId in URL path", http.StatusBadRequest)
		return
	}

	// --- Retrieve the model using the interface ---
	ctx := r.Context()
	foundModel, err := a.ModelStore.FindModelByID(ctx, modelID)
	if err != nil {
		log.Printf("DEBUG: Failed to find model '%s': %v", modelID, err) // Use DEBUG/INFO level
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve model", http.StatusInternalServerError)
		}
		return
	}
	// --- End Retrieve ---

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(foundModel); err != nil {
		log.Printf("ERROR: Failed to encode get model response: %v", err)
	}
}

// ListModels handles GET requests to /models
func (a *API) ListModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	modelsList, err := a.ModelStore.ListAllModels(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to list models: %v", err)
		http.Error(w, "Failed to retrieve models", http.StatusInternalServerError)
		return
	}

	// Optional: Sorting can still happen here if desired, DB usually handles it though
	sort.Slice(modelsList, func(i, j int) bool {
		return modelsList[i].ID < modelsList[j].ID
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(modelsList); err != nil {
		log.Printf("ERROR: Failed to encode list models response: %v", err)
	}
}

// DeleteModel handles DELETE requests to /models/{modelId}
func (a *API) DeleteModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "modelId")
	if modelID == "" {
		http.Error(w, "Missing modelId in URL path", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := a.ModelStore.DeleteModel(ctx, modelID)
	if err != nil {
		log.Printf("DEBUG: Failed to delete model '%s': %v", modelID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Deleted model: ID=%s", modelID)
	w.WriteHeader(http.StatusNoContent)
}

// UpdateModel handles PUT requests to /models/{modelId}
func (a *API) UpdateModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "modelId")
	if modelID == "" {
		http.Error(w, "Missing modelId in URL path", http.StatusBadRequest)
		return
	}

	var updatedModelData model.TwinModel
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&updatedModelData); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if updatedModelData.DisplayName == "" {
		http.Error(w, "Missing required field: displayName", http.StatusBadRequest)
		return
	}

	// Ensure the ID in the payload matches the URL path ID (optional but good practice)
	if updatedModelData.ID != "" && updatedModelData.ID != modelID {
		http.Error(w, "Model ID in payload does not match ID in URL", http.StatusBadRequest)
		return
	}
	updatedModelData.ID = modelID // Ensure the correct ID is set for the update operation

	// We set UpdatedAt here, but the DB trigger will overwrite it on successful update.
	updatedModelData.UpdatedAt = time.Now().UTC()

	ctx := r.Context()
	err := a.ModelStore.UpdateModel(ctx, &updatedModelData) // Pass pointer
	if err != nil {
		log.Printf("DEBUG: Failed to update model '%s': %v", modelID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Model not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update model", http.StatusInternalServerError)
		}
		return
	}

	// Since UpdateModel doesn't return the updated object, we need to fetch it again
	// to return the latest state (including potentially DB-generated timestamps)
	updatedModel, findErr := a.ModelStore.FindModelByID(ctx, modelID)
	if findErr != nil {
		log.Printf("ERROR: Failed to retrieve updated model '%s' after update: %v", modelID, findErr)
		http.Error(w, "Failed to retrieve model after update", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Updated model: ID=%s", updatedModel.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedModel); err != nil { // Return the re-fetched model
		log.Printf("ERROR: Failed to encode update model response: %v", err)
	}
}

// --- Health Check Handler ---
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Enhance health check to ping DB via the store interface if needed
	response := map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode health check response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
