-- sql/001_create_twin_models.sql

-- Enable UUID generation if not already enabled
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; -- Alternative if needed

CREATE TABLE IF NOT EXISTS twin_models (
    id VARCHAR(255) PRIMARY KEY, -- Using VARCHAR for flexibility (DTMI or UUID)
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    -- Placeholders for future structured data (JSONB is great here)
    -- properties JSONB,
    -- telemetry JSONB,
    -- commands JSONB,
    -- events JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Optional: Index on display_name if you query by it often
-- CREATE INDEX IF NOT EXISTS idx_twin_models_display_name ON twin_models(display_name);

-- Optional: Trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp ON twin_models; -- Drop existing trigger if recreating
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON twin_models
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();