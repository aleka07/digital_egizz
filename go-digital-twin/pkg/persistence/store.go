// pkg/persistence/store.go
package persistence

import (
	"context" // Use context for cancellation and deadlines
	// For checking specific persistence errors
	"time" // Need time for telemetry

	// Keep using standard errors or define custom ones
	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/model" // UPDATE THE PATH
)

// ModelStore defines the interface for persistence operations related to TwinModels.
type ModelStore interface {
	// Create stores a new TwinModel. Returns an error if the ID already exists or on DB failure.
	CreateModel(ctx context.Context, model *model.TwinModel) error

	// FindByID retrieves a TwinModel by its unique ID. Returns model.ErrNotFound if not found.
	FindModelByID(ctx context.Context, id string) (*model.TwinModel, error)

	// ListAll lists all stored TwinModels.
	ListAllModels(ctx context.Context) ([]*model.TwinModel, error)

	// Update modifies an existing TwinModel. Returns model.ErrNotFound if the model doesn't exist.
	UpdateModel(ctx context.Context, model *model.TwinModel) error

	// Delete removes a TwinModel by its ID. Returns model.ErrNotFound if not found.
	DeleteModel(ctx context.Context, id string) error

	// Close cleans up resources (e.g., database connections).
	Close() // No context needed for Close usually
}

// TwinStore defines the interface for persistence operations related to TwinInstances.
type TwinStore interface {
	// Create stores a new TwinInstance. Requires a valid ModelID.
	CreateTwin(ctx context.Context, twin *model.TwinInstance) error

	// FindByID retrieves a TwinInstance by its unique ID. Returns ErrNotFound if not found.
	FindTwinByID(ctx context.Context, id string) (*model.TwinInstance, error)

	// ListAll lists all stored TwinInstances. Add filtering/pagination later.
	ListAllTwins(ctx context.Context) ([]*model.TwinInstance, error)

	// ListByModel lists twins associated with a specific model ID.
	ListTwinsByModel(ctx context.Context, modelID string) ([]*model.TwinInstance, error)

	// Update modifies mutable fields of an existing TwinInstance (e.g., properties, tags).
	// This might be split into more granular updates later (UpdateProperties, UpdateTags).
	UpdateTwin(ctx context.Context, twin *model.TwinInstance) error

	// UpdateReportedProperties specifically updates the reported properties field.
	UpdateReportedProperties(ctx context.Context, id string, properties map[string]interface{}) error

	// UpdateDesiredProperties specifically updates the desired properties field.
	UpdateDesiredProperties(ctx context.Context, id string, properties map[string]interface{}) error

	// UpdateTags specifically updates the tags field.
	UpdateTags(ctx context.Context, id string, tags map[string]string) error

	// Delete removes a TwinInstance by its ID. Returns ErrNotFound if not found.
	DeleteTwin(ctx context.Context, id string) error

	// Close cleans up resources (can reuse ModelStore's Close if combined).
	// Close() // Only needed if TwinStore is a separate struct with its own resources
}

// TelemetryRecord represents a single time-series data point.
// Using a struct makes it easier to handle multiple value types.
type TelemetryRecord struct {
	Timestamp    time.Time `json:"ts"`
	TwinID       string    `json:"-"` // Usually known from context, not needed in JSON response body
	Name         string    `json:"name"`
	NumericValue *float64  `json:"numValue,omitempty"` // Pointer to distinguish null from 0
	StringValue  *string   `json:"stringValue,omitempty"`
	BooleanValue *bool     `json:"boolValue,omitempty"`
	// JSONValue    interface{} `json:"jsonValue,omitempty"` // Add if using value_jsonb
}

// TimeSeriesStore defines the interface for persistence operations for telemetry data.
type TimeSeriesStore interface {
	// WriteTelemetry stores a single telemetry record.
	WriteTelemetry(ctx context.Context, twinID string, record *TelemetryRecord) error

	// WriteBatchTelemetry stores multiple telemetry records efficiently. (Implement later if needed)
	// WriteBatchTelemetry(ctx context.Context, twinID string, records []*TelemetryRecord) error

	// QueryTelemetryHistory retrieves historical telemetry for a specific twin and metric name
	// within a given time range. Add aggregation, downsampling options later.
	QueryTelemetryHistory(ctx context.Context, twinID string, name string, start time.Time, end time.Time, descending bool, limit uint) ([]*TelemetryRecord, error)

	// QueryLatest retrieves the most recent telemetry record(s) for a twin.
	// Can filter by name or get latest for all names.
	QueryLatestTelemetry(ctx context.Context, twinID string, names []string) (map[string]*TelemetryRecord, error) // Map of name -> latest record

	// Close cleans up resources (can reuse ModelStore's Close if combined).
	// Close()
}

// Combined Store Interface (Optional but convenient)
// Allows API handlers to depend on a single store object if implementation is combined.
type Store interface {
	ModelStore
	TwinStore
	TimeSeriesStore // Add the new interface
	// Add TimeSeriesStore later
	Close() // Single Close method
}

// model-802fddd7-8818-4080-9bd9-1fa7ec392d73 - B
// model-1e04c2c2-c4b5-48ff-8ecc-cac8399c7fc6 - A
