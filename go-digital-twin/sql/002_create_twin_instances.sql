      
-- sql/002_create_twin_instances.sql

CREATE TABLE IF NOT EXISTS twin_instances (
    id VARCHAR(255) PRIMARY KEY,         -- Unique instance ID (e.g., UUID)
    model_id VARCHAR(255) NOT NULL,    -- Foreign key to twin_models table

    -- Store properties as JSONB for flexibility
    -- We can query inside JSONB efficiently if needed
    reported_properties JSONB DEFAULT '{}'::jsonb,
    desired_properties JSONB DEFAULT '{}'::jsonb,

    -- Store tags as JSONB (key-value pairs)
    tags JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key constraint (optional but recommended for data integrity)
    -- Ensure model_id exists in the twin_models table
    CONSTRAINT fk_model
        FOREIGN KEY(model_id)
        REFERENCES twin_models(id)
        ON DELETE RESTRICT -- Prevent deleting a model if instances still use it
        -- ON DELETE CASCADE -- Alternative: Delete instances if model is deleted
        -- ON DELETE SET NULL -- Alternative: Set model_id to NULL if model is deleted
);

-- Indexes for frequent lookups
CREATE INDEX IF NOT EXISTS idx_twin_instances_model_id ON twin_instances(model_id);

-- Indexes for querying properties/tags (GIN indexes are good for JSONB)
-- Example: Index specific keys if queried often, or index the whole document.
CREATE INDEX IF NOT EXISTS idx_twin_instances_tags ON twin_instances USING GIN (tags);
-- CREATE INDEX IF NOT EXISTS idx_twin_instances_reported_prop_location ON twin_instances USING GIN ((reported_properties -> 'location')); -- Example

-- Use the same trigger function for updated_at
-- Make sure the trigger function 'trigger_set_timestamp' was created by 001_create_twin_models.sql
-- If not, uncomment and run this block first:
/*
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
*/

DROP TRIGGER IF EXISTS set_timestamp ON twin_instances; -- Drop existing trigger if recreating
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON twin_instances
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp(); -- Reuse function

    