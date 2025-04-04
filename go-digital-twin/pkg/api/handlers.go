// pkg/api/handlers.go
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http" // For parsing query parameters
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/model"
	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/persistence"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// --- API Struct (Accepts combined Store interface) ---
type API struct {
	Store persistence.Store // Use the combined Store interface
}

// NewAPI creates a new API handler structure.
func NewAPI(store persistence.Store) *API { // Accept combined Store interface
	return &API{
		Store: store,
	}
}

// --- Model Handlers ---

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
	ctx := r.Context()                         // Get context from request
	err := a.Store.CreateModel(ctx, &newModel) // Pass pointer
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
	foundModel, err := a.Store.FindModelByID(ctx, modelID)
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
	modelsList, err := a.Store.ListAllModels(ctx)
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
	err := a.Store.DeleteModel(ctx, modelID)
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
	err := a.Store.UpdateModel(ctx, &updatedModelData) // Pass pointer
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
	updatedModel, findErr := a.Store.FindModelByID(ctx, modelID)
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

// --- Twin Instance Handlers ---

// CreateTwin handles POST requests to /twins
func (a *API) CreateTwin(w http.ResponseWriter, r *http.Request) {
	var reqBody struct { // Use a temporary struct for the request body
		ID           string                 `json:"id"` // Allow client to suggest ID, but generate if empty
		ModelID      string                 `json:"modelId"`
		DesiredProps map[string]interface{} `json:"desiredProperties"`
		Tags         map[string]string      `json:"tags"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// --- Validation ---
	if reqBody.ModelID == "" {
		http.Error(w, "Missing required field: modelId", http.StatusBadRequest)
		return
	}

	// Check if the specified Model exists
	ctx := r.Context()
	_, err := a.Store.FindModelByID(ctx, reqBody.ModelID)
	if err != nil {
		if errors.Is(err, persistence.ErrNotFound) {
			// Use BadRequest because the client provided an invalid reference
			http.Error(w, fmt.Sprintf("Referenced modelId '%s' not found", reqBody.ModelID), http.StatusBadRequest)
		} else {
			log.Printf("ERROR: Failed to check model existence: %v", err)
			http.Error(w, "Failed to validate modelId", http.StatusInternalServerError)
		}
		return
	}

	// Generate ID if not provided
	twinID := reqBody.ID
	if twinID == "" {
		twinID = "twin-" + uuid.NewString() // Prefix + UUID
	}

	now := time.Now().UTC()
	newTwin := &model.TwinInstance{
		ID:                 twinID,
		ModelID:            reqBody.ModelID,
		ReportedProperties: make(map[string]interface{}), // Initialize as empty
		DesiredProperties:  reqBody.DesiredProps,         // Use provided desired props
		Tags:               reqBody.Tags,                 // Use provided tags
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Ensure maps are non-nil before passing to store (store also handles this, but good practice)
	if newTwin.DesiredProperties == nil {
		newTwin.DesiredProperties = make(map[string]interface{})
	}
	if newTwin.Tags == nil {
		newTwin.Tags = make(map[string]string)
	}

	// --- Store the twin ---
	err = a.Store.CreateTwin(ctx, newTwin)
	if err != nil {
		log.Printf("ERROR: Failed to create twin: %v", err)
		if errors.Is(err, persistence.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			// Don't need to re-check FK error here as we validated modelId above
			http.Error(w, "Failed to create twin", http.StatusInternalServerError)
		}
		return
	}
	// --- End Store ---

	log.Printf("INFO: Created twin: ID=%s, ModelID=%s", newTwin.ID, newTwin.ModelID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newTwin); err != nil {
		log.Printf("ERROR: Failed to encode create twin response: %v", err)
	}
}

// GetTwin handles GET requests to /twins/{twinId}
func (a *API) GetTwin(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	twin, err := a.Store.FindTwinByID(ctx, twinID)
	if err != nil {
		log.Printf("DEBUG: Failed to find twin '%s': %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Twin not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve twin", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(twin); err != nil {
		log.Printf("ERROR: Failed to encode get twin response: %v", err)
	}
}

// ListTwins handles GET requests to /twins
func (a *API) ListTwins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Basic Filtering (Example: by modelId)
	modelIdQuery := r.URL.Query().Get("modelId") // Get "?modelId=..." query param

	var twinsList []*model.TwinInstance
	var err error

	if modelIdQuery != "" {
		// Optional: Check if model actually exists first? Maybe not necessary for List.
		twinsList, err = a.Store.ListTwinsByModel(ctx, modelIdQuery)
		log.Printf("INFO: Listing twins for modelId: %s", modelIdQuery)
	} else {
		twinsList, err = a.Store.ListAllTwins(ctx)
		log.Printf("INFO: Listing all twins")
	}

	if err != nil {
		log.Printf("ERROR: Failed to list twins: %v", err)
		http.Error(w, "Failed to retrieve twins", http.StatusInternalServerError)
		return
	}

	// Ensure non-nil slice is returned even if empty
	if twinsList == nil {
		twinsList = make([]*model.TwinInstance, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(twinsList); err != nil {
		log.Printf("ERROR: Failed to encode list twins response: %v", err)
	}
}

// DeleteTwin handles DELETE requests to /twins/{twinId}
func (a *API) DeleteTwin(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := a.Store.DeleteTwin(ctx, twinID)
	if err != nil {
		log.Printf("DEBUG: Failed to delete twin '%s': %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Twin not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete twin", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Deleted twin: ID=%s", twinID)
	w.WriteHeader(http.StatusNoContent)
}

// UpdateTwin handles PUT requests to /twins/{twinId}
// This replaces ModelID, DesiredProperties, and Tags based on request body.
// Caution: ReportedProperties are NOT updated via this endpoint.
func (a *API) UpdateTwin(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	// 1. Fetch the existing twin to get CreatedAt and ReportedProperties
	ctx := r.Context()
	existingTwin, err := a.Store.FindTwinByID(ctx, twinID)
	if err != nil {
		log.Printf("DEBUG: Failed to find twin '%s' for update: %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Twin not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve twin for update", http.StatusInternalServerError)
		}
		return
	}

	// 2. Decode request body containing fields to update
	var reqBody struct {
		ModelID      *string                `json:"modelId"` // Use pointers to detect if field is present
		DesiredProps map[string]interface{} `json:"desiredProperties"`
		Tags         map[string]string      `json:"tags"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Prepare the updated TwinInstance struct
	updatedTwin := &model.TwinInstance{
		ID:                 twinID,                          // Keep original ID
		ModelID:            existingTwin.ModelID,            // Keep original model unless provided
		ReportedProperties: existingTwin.ReportedProperties, // IMPORTANT: Keep existing reported props
		DesiredProperties:  existingTwin.DesiredProperties,  // Keep existing desired unless provided
		Tags:               existingTwin.Tags,               // Keep existing tags unless provided
		CreatedAt:          existingTwin.CreatedAt,          // Keep original CreatedAt
		UpdatedAt:          time.Now().UTC(),                // Set update time
	}

	// Apply updates from request body if fields were provided
	if reqBody.ModelID != nil {
		// Validate the new model ID if it's being changed
		_, err := a.Store.FindModelByID(ctx, *reqBody.ModelID)
		if err != nil {
			if errors.Is(err, persistence.ErrNotFound) {
				http.Error(w, fmt.Sprintf("Referenced modelId '%s' not found", *reqBody.ModelID), http.StatusBadRequest)
			} else {
				log.Printf("ERROR: Failed to check new model existence: %v", err)
				http.Error(w, "Failed to validate new modelId", http.StatusInternalServerError)
			}
			return
		}
		updatedTwin.ModelID = *reqBody.ModelID
	}
	if reqBody.DesiredProps != nil { // Check if the key was present in JSON, even if value is null/empty
		updatedTwin.DesiredProperties = reqBody.DesiredProps
	}
	if reqBody.Tags != nil {
		updatedTwin.Tags = reqBody.Tags
	}

	// 4. Store the updated twin using the general UpdateTwin method
	err = a.Store.UpdateTwin(ctx, updatedTwin)
	if err != nil {
		log.Printf("ERROR: Failed to update twin '%s': %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			// Should not happen if FindTwinByID succeeded, but check anyway
			http.Error(w, "Twin not found during update", http.StatusNotFound)
		} else if errors.Is(err, persistence.ErrConflict) { // e.g., FK violation if modelId changed
			http.Error(w, err.Error(), http.StatusBadRequest) // Or Conflict? Bad Request seems better for FK.
		} else {
			http.Error(w, "Failed to update twin", http.StatusInternalServerError)
		}
		return
	}

	// 5. Return the fully updated twin (re-fetch might be needed if DB changes things)
	// Since UpdateTwin doesn't return the object, re-fetch for consistency
	finalTwin, findErr := a.Store.FindTwinByID(ctx, twinID)
	if findErr != nil {
		log.Printf("ERROR: Failed to retrieve updated twin '%s' after PUT: %v", twinID, findErr)
		http.Error(w, "Failed to retrieve twin after update", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Updated twin (PUT): ID=%s", finalTwin.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(finalTwin); err != nil {
		log.Printf("ERROR: Failed to encode update twin response: %v", err)
	}
}

// --- Specific Update Handlers ---

// UpdateTwinDesiredProperties handles PUT requests to /twins/{twinId}/properties/desired
func (a *API) UpdateTwinDesiredProperties(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	var props map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&props); err != nil {
		http.Error(w, "Invalid request payload (expecting JSON object): "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if props == nil {
		props = make(map[string]interface{}) // Ensure non-nil map for update
	}

	ctx := r.Context()
	err := a.Store.UpdateDesiredProperties(ctx, twinID, props)
	if err != nil {
		log.Printf("ERROR: Failed to update desired properties for twin '%s': %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Twin not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update desired properties", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the updated twin to return the full object
	updatedTwin, findErr := a.Store.FindTwinByID(ctx, twinID)
	if findErr != nil {
		log.Printf("ERROR: Failed to retrieve twin '%s' after desired prop update: %v", twinID, findErr)
		http.Error(w, "Failed to retrieve twin after update", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Updated desired properties for twin: ID=%s", twinID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedTwin); err != nil {
		log.Printf("ERROR: Failed to encode update desired props response: %v", err)
	}
}

// UpdateTwinTags handles PUT requests to /twins/{twinId}/tags
func (a *API) UpdateTwinTags(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	var tags map[string]string
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&tags); err != nil {
		http.Error(w, "Invalid request payload (expecting JSON object with string values): "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if tags == nil {
		tags = make(map[string]string) // Ensure non-nil map for update
	}

	ctx := r.Context()
	err := a.Store.UpdateTags(ctx, twinID, tags)
	if err != nil {
		log.Printf("ERROR: Failed to update tags for twin '%s': %v", twinID, err)
		if errors.Is(err, persistence.ErrNotFound) {
			http.Error(w, "Twin not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update tags", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the updated twin to return the full object
	updatedTwin, findErr := a.Store.FindTwinByID(ctx, twinID)
	if findErr != nil {
		log.Printf("ERROR: Failed to retrieve twin '%s' after tags update: %v", twinID, findErr)
		http.Error(w, "Failed to retrieve twin after update", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Updated tags for twin: ID=%s", twinID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedTwin); err != nil {
		log.Printf("ERROR: Failed to encode update tags response: %v", err)
	}
}

// --- Telemetry Handlers ---

// GetTelemetryHistory handles GET requests to /twins/{twinId}/telemetry/{telemetryName}/history
func (a *API) GetTelemetryHistory(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	telemetryName := chi.URLParam(r, "telemetryName") // Get name from path

	if twinID == "" || telemetryName == "" {
		http.Error(w, "Missing twinId or telemetryName in URL path", http.StatusBadRequest)
		return
	}

	// --- Parse Query Parameters ---
	query := r.URL.Query()

	// Default time range (e.g., last hour)
	defaultEnd := time.Now().UTC()
	defaultStart := defaultEnd.Add(-1 * time.Hour)

	// Parse start time (RFC3339 format, e.g., 2023-10-27T10:00:00Z)
	start, err := time.Parse(time.RFC3339, query.Get("start"))
	if err != nil || query.Get("start") == "" {
		start = defaultStart // Use default if missing or invalid
	}

	// Parse end time
	end, err := time.Parse(time.RFC3339, query.Get("end"))
	if err != nil || query.Get("end") == "" {
		end = defaultEnd // Use default if missing or invalid
	}

	// Ensure start is before end
	if start.After(end) {
		http.Error(w, "Invalid time range: start time must be before end time", http.StatusBadRequest)
		return
	}

	// Parse order (desc or asc)
	descending := strings.ToLower(query.Get("order")) == "desc"

	// Parse limit (positive integer)
	var limit uint = 0 // Default: no limit
	limitStr := query.Get("limit")
	if limitStr != "" {
		parsedLimit, err := strconv.ParseUint(limitStr, 10, 32) // Parse as uint
		if err == nil && parsedLimit > 0 {
			limit = uint(parsedLimit)
		} else {
			http.Error(w, "Invalid limit parameter: must be a positive integer", http.StatusBadRequest)
			return
		}
	}

	// --- Query the Store ---
	ctx := r.Context()
	records, err := a.Store.QueryTelemetryHistory(ctx, twinID, telemetryName, start, end, descending, limit)
	if err != nil {
		// Note: Don't return 404 if twin exists but has no telemetry in range.
		// The store method doesn't distinguish "twin not found" from "no data found".
		// We could add a separate check for twin existence if needed.
		log.Printf("ERROR: Failed to query telemetry history for twin '%s', name '%s': %v", twinID, telemetryName, err)
		http.Error(w, "Failed to retrieve telemetry history", http.StatusInternalServerError)
		return
	}

	// Ensure non-nil slice is returned even if empty
	if records == nil {
		records = make([]*persistence.TelemetryRecord, 0)
	}

	// --- Respond ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("ERROR: Failed to encode telemetry history response: %v", err)
	}
}

// GetLatestTelemetry handles GET requests to /twins/{twinId}/telemetry/latest
func (a *API) GetLatestTelemetry(w http.ResponseWriter, r *http.Request) {
	twinID := chi.URLParam(r, "twinId")
	if twinID == "" {
		http.Error(w, "Missing twinId in URL path", http.StatusBadRequest)
		return
	}

	// Optional: Filter by specific names provided in query param?
	// e.g., ?name=temperature&name=humidity
	namesFilter := r.URL.Query()["name"] // Gets slice of values for "name"

	// --- Query the Store ---
	ctx := r.Context()
	latestValues, err := a.Store.QueryLatestTelemetry(ctx, twinID, namesFilter)
	if err != nil {
		// Again, don't assume 404, check twin existence separately if needed.
		log.Printf("ERROR: Failed to query latest telemetry for twin '%s': %v", twinID, err)
		http.Error(w, "Failed to retrieve latest telemetry", http.StatusInternalServerError)
		return
	}

	// Ensure non-nil map is returned even if empty
	if latestValues == nil {
		latestValues = make(map[string]*persistence.TelemetryRecord)
	}

	// --- Respond ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(latestValues); err != nil { // Encode the map
		log.Printf("ERROR: Failed to encode latest telemetry response: %v", err)
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
