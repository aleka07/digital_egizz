// pkg/persistence/store.go
package persistence

import (
	"context" // Use context for cancellation and deadlines

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

// model-802fddd7-8818-4080-9bd9-1fa7ec392d73 - B
// model-1e04c2c2-c4b5-48ff-8ecc-cac8399c7fc6 - A
