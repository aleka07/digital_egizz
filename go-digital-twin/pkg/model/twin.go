// pkg/model/twin.go
package model

import "time" // We might need time later

// TwinModel defines the blueprint for a type of digital twin.
// It specifies the expected properties, telemetry, commands, etc.
type TwinModel struct {
	ID          string `json:"id" yaml:"id"`                                       // Unique identifier for the model (e.g., DTMI like "dtmi:com:example:thermostat;1")
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"` // User-friendly name
	Description string `json:"description,omitempty" yaml:"description,omitempty"` // Optional description

	// --- Placeholders for later ---
	// Properties map[string]PropertyDefinition `json:"properties,omitempty" yaml:"properties,omitempty"`
	// Telemetry  map[string]TelemetryDefinition `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	// Commands   map[string]CommandDefinition   `json:"commands,omitempty" yaml:"commands,omitempty"`
	// Events     map[string]EventDefinition     `json:"events,omitempty" yaml:"events,omitempty"`

	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"` // Timestamp of model creation
	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"` // Timestamp of last model update
}

// TwinInstance represents a specific digital twin based on a TwinModel.
// It holds the current state and identity of a real-world device/asset.
type TwinInstance struct {
	ID      string `json:"id" yaml:"id"`           // Unique instance ID (e.g., UUID)
	ModelID string `json:"modelId" yaml:"modelId"` // ID of the TwinModel this instance implements

	// --- Placeholders for later ---
	// ReportedProperties map[string]interface{} `json:"reportedProperties,omitempty"` // Last known state reported by the device
	// DesiredProperties  map[string]interface{} `json:"desiredProperties,omitempty"`  // Target state set by applications
	// Tags             map[string]string      `json:"tags,omitempty"`               // Metadata tags for querying/grouping

	CreatedAt time.Time `json:"createdAt"` // Timestamp of instance creation
	UpdatedAt time.Time `json:"updatedAt"` // Timestamp of last instance update (state change, etc.)
}

// --- Placeholder definitions for Properties, Telemetry, etc. ---
// We'll flesh these out in later steps when we implement model validation and state management.
/*
type PropertyDefinition struct {
    Name        string      `json:"name" yaml:"name"`
    Schema      string      `json:"schema" yaml:"schema"` // e.g., "string", "double", "boolean", "object"
    Writable    bool        `json:"writable" yaml:"writable"`
    Unit        string      `json:"unit,omitempty" yaml:"unit,omitempty"`
    Description string      `json:"description,omitempty" yaml:"description,omitempty"`
}

type TelemetryDefinition struct {
    Name        string      `json:"name" yaml:"name"`
    Schema      string      `json:"schema" yaml:"schema"`
    Unit        string      `json:"unit,omitempty" yaml:"unit,omitempty"`
    Description string      `json:"description,omitempty" yaml:"description,omitempty"`
}

// ... CommandDefinition, EventDefinition ...
*/
