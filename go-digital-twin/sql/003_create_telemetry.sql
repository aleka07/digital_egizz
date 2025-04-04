-- sql/003_create_telemetry.sql

-- Create regular PostgreSQL table first
CREATE TABLE IF NOT EXISTS telemetry (
    ts TIMESTAMPTZ NOT NULL,          -- Timestamp of the reading (must be TIMESTAMPTZ for Timescale)
    twin_id VARCHAR(255) NOT NULL,   -- ID of the twin emitting the telemetry
    name VARCHAR(255) NOT NULL,      -- Name of the telemetry field (e.g., "temperature", "pressure")

    -- Store the value flexibly. Using separate columns is often more efficient
    -- for querying/indexing than a single JSONB if you know the common types.
    value_numeric DOUBLE PRECISION,    -- For numeric values (float/integer)
    value_string TEXT,                -- For string values
    value_boolean BOOLEAN             -- For boolean values
    -- Add value_location geography/geometry if needed
    -- Add value_jsonb JSONB for complex/object values if needed
);

-- Create TimescaleDB hypertable, partitioning by time (ts column)
-- This converts the 'telemetry' table into a hypertable.
-- Adjust chunk_time_interval based on expected data volume and query patterns.
-- '7 days' is a common starting point.
SELECT create_hypertable('telemetry', 'ts', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

-- Create indexes for efficient querying (TimescaleDB automatically indexes the time column)
-- A composite index on (twin_id, name, ts) is crucial for typical lookups.
CREATE INDEX IF NOT EXISTS idx_telemetry_twin_name_ts ON telemetry (twin_id, name, ts DESC);
-- Optional: Index only twin_id and ts if you often query all telemetry for a twin in a time range
-- CREATE INDEX IF NOT EXISTS idx_telemetry_twin_ts ON telemetry (twin_id, ts DESC);
-- Optional: Index name separately if you query across twins for a specific telemetry name
-- CREATE INDEX IF NOT EXISTS idx_telemetry_name_ts ON telemetry (name, ts DESC);

-- Optional: Consider TimescaleDB Compression after some time
-- SELECT add_compression_policy('telemetry', INTERVAL '14 days');

-- Optional: Consider Data Retention / Downsampling later
-- SELECT add_retention_policy('telemetry', INTERVAL '90 days');