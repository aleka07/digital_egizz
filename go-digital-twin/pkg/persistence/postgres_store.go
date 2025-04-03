// pkg/persistence/postgres_store.go
package persistence

import (
	"context"
	"errors" // For standard errors
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn" // For type assertion on errors
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/model" // UPDATE THE PATH
)

// --- Define specific errors ---
// Consider moving these to a central errors package later (e.g., pkg/model/errors.go)
var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict / already exists") // For duplicate keys

// PostgresModelStore implements the ModelStore interface using PostgreSQL.
type PostgresModelStore struct {
	pool *pgxpool.Pool // Use a connection pool for efficiency
}

// NewPostgresModelStore creates a new PostgreSQL model store.
// It expects a DSN (Data Source Name) string, e.g.,
// "postgres://user:password@host:port/database?sslmode=disable"
func NewPostgresModelStore(ctx context.Context, dsn string) (*PostgresModelStore, error) {
	log.Printf("INFO: Connecting to PostgreSQL: %s", dsn) // Avoid logging password in real apps
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Close pool if ping fails
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("INFO: PostgreSQL connection established successfully.")
	return &PostgresModelStore{pool: pool}, nil
}

// Close closes the database connection pool.
func (s *PostgresModelStore) Close() {
	log.Println("INFO: Closing PostgreSQL connection pool.")
	s.pool.Close()
}

// CreateModel inserts a new model into the database.
func (s *PostgresModelStore) CreateModel(ctx context.Context, m *model.TwinModel) error {
	query := `
        INSERT INTO twin_models (id, display_name, description, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)`

	_, err := s.pool.Exec(ctx, query, m.ID, m.DisplayName, m.Description, m.CreatedAt, m.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate key)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 is unique_violation
			return fmt.Errorf("%w: model with ID '%s' already exists", ErrConflict, m.ID)
		}
		return fmt.Errorf("failed to insert model: %w", err)
	}
	return nil
}

// FindModelByID retrieves a model by its ID.
func (s *PostgresModelStore) FindModelByID(ctx context.Context, id string) (*model.TwinModel, error) {
	query := `
        SELECT id, display_name, description, created_at, updated_at
        FROM twin_models
        WHERE id = $1`

	m := &model.TwinModel{} // Pointer to scan into
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.DisplayName,
		&m.Description,
		&m.CreatedAt,
		&m.UpdatedAt,
		// Scan future JSONB fields here if added
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: model with ID '%s' not found", ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to find model by ID: %w", err)
	}
	return m, nil
}

// ListAllModels retrieves all models from the database.
func (s *PostgresModelStore) ListAllModels(ctx context.Context) ([]*model.TwinModel, error) {
	query := `
        SELECT id, display_name, description, created_at, updated_at
        FROM twin_models
        ORDER BY id ASC` // Consistent ordering

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		// Don't check for ErrNoRows here, Query returns it implicitly when Next() is false
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close() // Ensure rows are closed

	models := []*model.TwinModel{}
	for rows.Next() {
		m := &model.TwinModel{}
		err := rows.Scan(
			&m.ID,
			&m.DisplayName,
			&m.Description,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			// Log intermediate errors but try to continue if possible,
			// or return immediately depending on requirements.
			log.Printf("WARN: Failed to scan model row: %v", err)
			// Returning error here might be safer:
			// return nil, fmt.Errorf("failed to scan model row: %w", err)
			continue // Or return error
		}
		models = append(models, m)
	}

	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model rows: %w", err)
	}

	return models, nil
}

// UpdateModel updates an existing model in the database.
func (s *PostgresModelStore) UpdateModel(ctx context.Context, m *model.TwinModel) error {
	// Note: The trigger handles updated_at automatically.
	// We pass m.UpdatedAt here just to align with Create, but it will be ignored by DB on successful UPDATE.
	// Alternatively, omit updated_at from the SET clause if you prefer.
	query := `
        UPDATE twin_models
        SET display_name = $2, description = $3, updated_at = $4
        WHERE id = $1`

	cmdTag, err := s.pool.Exec(ctx, query, m.ID, m.DisplayName, m.Description, m.UpdatedAt)

	if err != nil {
		// Could potentially check for unique constraint violation on display_name if it were unique
		return fmt.Errorf("failed to update model: %w", err)
	}

	// Check if any row was actually updated
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: model with ID '%s' not found for update", ErrNotFound, m.ID)
	}

	return nil
}

// DeleteModel removes a model from the database by ID.
func (s *PostgresModelStore) DeleteModel(ctx context.Context, id string) error {
	query := `DELETE FROM twin_models WHERE id = $1`

	cmdTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	// Check if a row was actually deleted
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: model with ID '%s' not found for deletion", ErrNotFound, id)
	}

	return nil
}
