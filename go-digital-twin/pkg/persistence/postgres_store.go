// pkg/persistence/postgres_store.go
package persistence

import (
	"context"
	"encoding/json" // Needed for handling JSONB potentially
	"errors"        // For standard errors
	"fmt"
	"log"     // For formatting limit
	"strings" // For query building
	"time"    // Needed for updated_at timestamp in specific updates

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn" // For type assertion on errors
	"github.com/jackc/pgx/v5/pgtype" // For handling NULLable types like float8
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aleka07/digital_egizz/go-digital-twin/pkg/model" // UPDATE THE PATH
)

// --- Define specific errors ---
// Consider moving these to a central errors package later (e.g., pkg/model/errors.go)
var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict / already exists") // For duplicate keys

// --- Ensure PostgresModelStore implements the combined Store interface ---
var _ Store = (*PostgresModelStore)(nil) // Compile-time check

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

// --- TwinStore Methods ---

// scanTwin reads a twin instance from a pgx.Row or pgx.Rows object.
// Helper function to avoid repetition.
func scanTwin(scanner pgx.Row /* or pgx.Rows */) (*model.TwinInstance, error) {
	t := &model.TwinInstance{}
	// We need intermediary []byte slices for JSONB fields
	var reportedPropsBytes, desiredPropsBytes, tagsBytes []byte

	// Adjust Scan arguments based on the SELECT query order
	err := scanner.Scan(
		&t.ID,
		&t.ModelID,
		&reportedPropsBytes, // Scan JSONB into []byte first
		&desiredPropsBytes,  // Scan JSONB into []byte first
		&tagsBytes,          // Scan JSONB into []byte first
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err // Return scan error directly
	}

	// Unmarshal JSONB bytes into Go maps
	if reportedPropsBytes != nil {
		if err := json.Unmarshal(reportedPropsBytes, &t.ReportedProperties); err != nil {
			return nil, fmt.Errorf("failed to unmarshal reported_properties: %w", err)
		}
	} else {
		t.ReportedProperties = make(map[string]interface{}) // Ensure map is non-nil
	}

	if desiredPropsBytes != nil {
		if err := json.Unmarshal(desiredPropsBytes, &t.DesiredProperties); err != nil {
			return nil, fmt.Errorf("failed to unmarshal desired_properties: %w", err)
		}
	} else {
		t.DesiredProperties = make(map[string]interface{}) // Ensure map is non-nil
	}

	if tagsBytes != nil {
		if err := json.Unmarshal(tagsBytes, &t.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	} else {
		t.Tags = make(map[string]string) // Ensure map is non-nil
	}

	return t, nil
}

// CreateTwin inserts a new twin instance.
func (s *PostgresModelStore) CreateTwin(ctx context.Context, twin *model.TwinInstance) error {
	query := `
        INSERT INTO twin_instances
            (id, model_id, reported_properties, desired_properties, tags, created_at, updated_at)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7)`

	// Marshal maps to JSON bytes for storing in JSONB columns
	// Handle nil maps gracefully, default to '{}'
	reportedPropsJSON, err := json.Marshal(twin.ReportedProperties)
	if err != nil || twin.ReportedProperties == nil {
		reportedPropsJSON = []byte("{}")
		if err != nil { // Log original marshal error if it occurred
			log.Printf("WARN: Could not marshal reported properties for twin %s: %v. Defaulting to {}", twin.ID, err)
		}
	}

	desiredPropsJSON, err := json.Marshal(twin.DesiredProperties)
	if err != nil || twin.DesiredProperties == nil {
		desiredPropsJSON = []byte("{}")
		if err != nil {
			log.Printf("WARN: Could not marshal desired properties for twin %s: %v. Defaulting to {}", twin.ID, err)
		}
	}

	tagsJSON, err := json.Marshal(twin.Tags)
	if err != nil || twin.Tags == nil {
		tagsJSON = []byte("{}")
		if err != nil {
			log.Printf("WARN: Could not marshal tags for twin %s: %v. Defaulting to {}", twin.ID, err)
		}
	}

	_, err = s.pool.Exec(ctx, query,
		twin.ID,
		twin.ModelID,
		reportedPropsJSON,
		desiredPropsJSON,
		tagsJSON,
		twin.CreatedAt,
		twin.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation (PK)
				return fmt.Errorf("%w: twin instance with ID '%s' already exists", ErrConflict, twin.ID)
			case "23503": // foreign_key_violation (model_id doesn't exist)
				return fmt.Errorf("%w: model with ID '%s' not found", ErrNotFound, twin.ModelID) // Treat FK violation as NotFound for the model
			}
		}
		return fmt.Errorf("failed to insert twin instance: %w", err)
	}
	return nil
}

// FindTwinByID retrieves a twin instance by ID.
func (s *PostgresModelStore) FindTwinByID(ctx context.Context, id string) (*model.TwinInstance, error) {
	query := `
        SELECT id, model_id, reported_properties, desired_properties, tags, created_at, updated_at
        FROM twin_instances
        WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, id)
	twin, err := scanTwin(row) // Use the helper

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: twin instance with ID '%s' not found", ErrNotFound, id)
		}
		// Handle scan errors (including JSON unmarshal errors from scanTwin)
		return nil, fmt.Errorf("failed to find or scan twin instance by ID: %w", err)
	}
	return twin, nil
}

// ListAllTwins retrieves all twin instances. Use LIMIT/OFFSET for pagination in real apps.
func (s *PostgresModelStore) ListAllTwins(ctx context.Context) ([]*model.TwinInstance, error) {
	query := `
        SELECT id, model_id, reported_properties, desired_properties, tags, created_at, updated_at
        FROM twin_instances
        ORDER BY id ASC` // Or ORDER BY created_at, etc.

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query twin instances: %w", err)
	}
	defer rows.Close()

	twins := []*model.TwinInstance{}
	for rows.Next() {
		twin, err := scanTwin(rows) // Use helper
		if err != nil {
			// Log intermediate errors but try to continue or return early
			log.Printf("WARN: Failed to scan twin instance row during ListAll: %v", err)
			continue // Or return nil, fmt.Errorf("failed to scan twin instance row: %w", err)
		}
		twins = append(twins, twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin instance rows: %w", err)
	}

	return twins, nil
}

// ListTwinsByModel retrieves twins filtered by model ID.
func (s *PostgresModelStore) ListTwinsByModel(ctx context.Context, modelID string) ([]*model.TwinInstance, error) {
	query := `
        SELECT id, model_id, reported_properties, desired_properties, tags, created_at, updated_at
        FROM twin_instances
        WHERE model_id = $1
        ORDER BY id ASC`

	rows, err := s.pool.Query(ctx, query, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to query twin instances by model ID: %w", err)
	}
	defer rows.Close()

	twins := []*model.TwinInstance{}
	for rows.Next() {
		twin, err := scanTwin(rows) // Use helper
		if err != nil {
			log.Printf("WARN: Failed to scan twin instance row during ListByModel: %v", err)
			continue
		}
		twins = append(twins, twin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating twin instance rows by model: %w", err)
	}

	// It's okay to return an empty slice if no twins match the model ID
	return twins, nil
}

// UpdateTwin updates mutable fields. Caution: Overwrites entire JSONB fields.
// Consider using more granular JSONB update functions in SQL for partial updates if needed.
func (s *PostgresModelStore) UpdateTwin(ctx context.Context, twin *model.TwinInstance) error {
	query := `
        UPDATE twin_instances
        SET
            model_id = $2, -- Allow changing model? Maybe not desirable. Decide based on requirements.
            reported_properties = $3,
            desired_properties = $4,
            tags = $5,
            updated_at = $6 -- Pass explicitly, trigger will handle it anyway
        WHERE id = $1`

	// Marshal JSON fields
	reportedPropsJSON, err := json.Marshal(twin.ReportedProperties)
	if err != nil {
		reportedPropsJSON = []byte("{}")
	}
	desiredPropsJSON, err := json.Marshal(twin.DesiredProperties)
	if err != nil {
		desiredPropsJSON = []byte("{}")
	}
	tagsJSON, err := json.Marshal(twin.Tags)
	if err != nil {
		tagsJSON = []byte("{}")
	}

	cmdTag, err := s.pool.Exec(ctx, query,
		twin.ID,
		twin.ModelID, // Be careful if allowing model changes
		reportedPropsJSON,
		desiredPropsJSON,
		tagsJSON,
		twin.UpdatedAt, // Pass timestamp
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // FK violation if changing model_id to non-existent one
			return fmt.Errorf("%w: model with ID '%s' not found", ErrNotFound, twin.ModelID)
		}
		return fmt.Errorf("failed to update twin instance: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: twin instance with ID '%s' not found for update", ErrNotFound, twin.ID)
	}
	return nil
}

// updateTwinJSONField provides a helper for updating specific JSONB fields
func (s *PostgresModelStore) updateTwinJSONField(ctx context.Context, id string, fieldName string, data interface{}) error {
	// Marshal the data to JSON bytes
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Handle nil map or other marshal errors
		if data == nil {
			jsonData = []byte("{}")
		} else {
			return fmt.Errorf("failed to marshal %s data for twin '%s': %w", fieldName, id, err)
		}
	}

	// Use fmt.Sprintf carefully or use a more structured query builder
	// Ensure fieldName is safe (not from user input directly in the query string)
	query := fmt.Sprintf(`
        UPDATE twin_instances
        SET %s = $2, updated_at = $3
        WHERE id = $1`, fieldName) // fieldName is safe here as it's controlled internally

	cmdTag, err := s.pool.Exec(ctx, query, id, jsonData, time.Now().UTC())

	if err != nil {
		return fmt.Errorf("failed to update twin instance %s field: %w", fieldName, err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: twin instance with ID '%s' not found for %s update", ErrNotFound, id, fieldName)
	}
	return nil
}

// UpdateReportedProperties updates only the reported_properties field.
func (s *PostgresModelStore) UpdateReportedProperties(ctx context.Context, id string, properties map[string]interface{}) error {
	return s.updateTwinJSONField(ctx, id, "reported_properties", properties)
}

// UpdateDesiredProperties updates only the desired_properties field.
func (s *PostgresModelStore) UpdateDesiredProperties(ctx context.Context, id string, properties map[string]interface{}) error {
	return s.updateTwinJSONField(ctx, id, "desired_properties", properties)
}

// UpdateTags updates only the tags field.
func (s *PostgresModelStore) UpdateTags(ctx context.Context, id string, tags map[string]string) error {
	return s.updateTwinJSONField(ctx, id, "tags", tags)
}

// DeleteTwin removes a twin instance by ID.
func (s *PostgresModelStore) DeleteTwin(ctx context.Context, id string) error {
	query := `DELETE FROM twin_instances WHERE id = $1`
	cmdTag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete twin instance: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: twin instance with ID '%s' not found for deletion", ErrNotFound, id)
	}
	return nil
}

// --- TimeSeriesStore Methods ---

// WriteTelemetry stores a single telemetry record.
func (s *PostgresModelStore) WriteTelemetry(ctx context.Context, twinID string, record *TelemetryRecord) error {
	query := `
        INSERT INTO telemetry (ts, twin_id, name, value_numeric, value_string, value_boolean)
        VALUES ($1, $2, $3, $4, $5, $6)`

	// Use pgtype equivalents for pointers to handle NULLs correctly
	var numVal pgtype.Float8
	if record.NumericValue != nil {
		numVal = pgtype.Float8{Float64: *record.NumericValue, Valid: true}
	}
	var strVal pgtype.Text
	if record.StringValue != nil {
		strVal = pgtype.Text{String: *record.StringValue, Valid: true}
	}
	var boolVal pgtype.Bool
	if record.BooleanValue != nil {
		boolVal = pgtype.Bool{Bool: *record.BooleanValue, Valid: true}
	}

	_, err := s.pool.Exec(ctx, query,
		record.Timestamp,
		twinID, // Pass twinID explicitly
		record.Name,
		numVal,  // Pass pgtype value
		strVal,  // Pass pgtype value
		boolVal, // Pass pgtype value
	)

	if err != nil {
		// Specific errors unlikely here unless DB is down or schema mismatch
		return fmt.Errorf("failed to insert telemetry record: %w", err)
	}
	return nil
}

// QueryTelemetryHistory retrieves historical telemetry data.
func (s *PostgresModelStore) QueryTelemetryHistory(ctx context.Context, twinID string, name string, start time.Time, end time.Time, descending bool, limit uint) ([]*TelemetryRecord, error) {
	// Base query
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
        SELECT ts, name, value_numeric, value_string, value_boolean
        FROM telemetry
        WHERE twin_id = $1 AND name = $2 AND ts >= $3 AND ts <= $4 `) // Arguments: twinID, name, start, end

	// Add ordering
	if descending {
		queryBuilder.WriteString("ORDER BY ts DESC ")
	} else {
		queryBuilder.WriteString("ORDER BY ts ASC ")
	}

	// Add limit - use $5 placeholder
	args := []interface{}{twinID, name, start, end}
	if limit > 0 {
		queryBuilder.WriteString("LIMIT $5")
		args = append(args, limit)
	}

	// Execute query
	rows, err := s.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query telemetry history: %w", err)
	}
	defer rows.Close()

	// Scan results
	records := []*TelemetryRecord{}
	for rows.Next() {
		rec := &TelemetryRecord{TwinID: twinID} // Pre-fill known fields
		// Use pgtype vars to scan potentially NULL values
		var numVal pgtype.Float8
		var strVal pgtype.Text
		var boolVal pgtype.Bool

		err := rows.Scan(
			&rec.Timestamp,
			&rec.Name,
			&numVal,
			&strVal,
			&boolVal,
		)
		if err != nil {
			log.Printf("WARN: Failed to scan telemetry row: %v", err)
			continue // Or return error
		}

		// Convert pgtype back to pointers if valid
		if numVal.Valid {
			rec.NumericValue = &numVal.Float64
		}
		if strVal.Valid {
			rec.StringValue = &strVal.String
		}
		if boolVal.Valid {
			rec.BooleanValue = &boolVal.Bool
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating telemetry history rows: %w", err)
	}

	return records, nil
}

// QueryLatestTelemetry retrieves the most recent telemetry value for specified names.
func (s *PostgresModelStore) QueryLatestTelemetry(ctx context.Context, twinID string, names []string) (map[string]*TelemetryRecord, error) {
	if len(names) == 0 {
		// Maybe query all names? Or return error? Let's query all for now.
		// Alternatively: return make(map[string]*TelemetryRecord), nil
	}

	// Use TimescaleDB's last() function for efficiency
	// SELECT last(column, time_column) FROM hypertable WHERE ... GROUP BY ...;
	var queryBuilder strings.Builder
	args := []interface{}{twinID} // Start with twinID as $1

	queryBuilder.WriteString(`
        SELECT
            name,
            last(ts, ts) as last_ts,
            last(value_numeric, ts) as last_num,
            last(value_string, ts) as last_str,
            last(value_boolean, ts) as last_bool
        FROM telemetry
        WHERE twin_id = $1 `)

	// Add filtering by name if specific names are provided
	if len(names) > 0 {
		queryBuilder.WriteString("AND name = ANY($2) ") // Use ANY($2) with a string slice argument
		args = append(args, names)
	}

	queryBuilder.WriteString("GROUP BY name ORDER BY name")

	rows, err := s.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest telemetry: %w", err)
	}
	defer rows.Close()

	latestValues := make(map[string]*TelemetryRecord)
	for rows.Next() {
		rec := &TelemetryRecord{TwinID: twinID} // Pre-fill known fields
		// Use pgtype vars to scan potentially NULL values from last() aggregate
		var lastTs pgtype.Timestamptz
		var numVal pgtype.Float8
		var strVal pgtype.Text
		var boolVal pgtype.Bool

		err := rows.Scan(
			&rec.Name,
			&lastTs,
			&numVal,
			&strVal,
			&boolVal,
		)
		if err != nil {
			log.Printf("WARN: Failed to scan latest telemetry row: %v", err)
			continue // Or return error
		}

		if !lastTs.Valid {
			continue
		} // Skip if no timestamp found for this group

		rec.Timestamp = lastTs.Time

		// Convert pgtype back to pointers if valid
		if numVal.Valid {
			rec.NumericValue = &numVal.Float64
		}
		if strVal.Valid {
			rec.StringValue = &strVal.String
		}
		if boolVal.Valid {
			rec.BooleanValue = &boolVal.Bool
		}

		latestValues[rec.Name] = rec
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating latest telemetry rows: %w", err)
	}

	return latestValues, nil
}
