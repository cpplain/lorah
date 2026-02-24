// Package schema generates documentation-oriented JSON schema from the
// agent harness configuration structs. The schema is hand-crafted to
// match the structure of HarnessConfig and its nested types.
package lorah

// FieldInfo describes a single configuration field.
type FieldInfo struct {
	Description string               `json:"description,omitempty"`
	Type        string               `json:"type,omitempty"`
	Default     any                  `json:"default,omitempty"`
	Enum        []string             `json:"enum,omitempty"`
	Options     []string             `json:"options,omitempty"`
	Fields      map[string]FieldInfo `json:"fields,omitempty"`
	Items       *FieldInfo           `json:"items,omitempty"`
}

// Schema is the top-level schema map (field name → FieldInfo).
type Schema map[string]FieldInfo

// GenerateSchema returns the full configuration schema for HarnessConfig.
// The schema describes the complete configuration structure in config.json.
func GenerateSchema() Schema {
	return Schema{
		"harness": {
			Description: "Harness-specific settings for the agent loop",
			Type:        "object",
			Fields: map[string]FieldInfo{
				"max-iterations": {
					Description: "Maximum total sessions before stopping (null = unlimited)",
					Type:        "integer",
				},
				"auto-continue-delay": {
					Description: "Delay in seconds before auto-continuing to next session",
					Type:        "integer",
					Default:     float64(3),
				},
				"error-recovery": {
					Description: "Error recovery and circuit breaker configuration",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"max-consecutive-errors": {
							Description: "Maximum consecutive session errors before stopping",
							Type:        "integer",
							Default:     float64(5),
						},
						"initial-backoff-seconds": {
							Description: "Initial backoff delay after first error (seconds)",
							Type:        "number",
							Default:     float64(5),
						},
						"max-backoff-seconds": {
							Description: "Maximum backoff delay (capped exponential backoff)",
							Type:        "number",
							Default:     float64(120),
						},
						"backoff-multiplier": {
							Description: "Multiplier for exponential backoff",
							Type:        "number",
							Default:     float64(2),
						},
						"max-error-message-length": {
							Description: "Maximum length of error messages to include in state",
							Type:        "integer",
							Default:     float64(2000),
						},
					},
				},
			},
		},
		"claude": {
			Description: "Claude CLI configuration (passthrough to Claude CLI - lorah does not validate). Any flag or setting supported by Claude CLI can be specified here.",
			Type:        "object",
			Fields: map[string]FieldInfo{
				"flags": {
					Description: "CLI flags passed as command-line arguments. Keys must be explicit flag names (e.g., '--max-turns'). Values are serialized and passed to Claude CLI. Use null for boolean flags without values.",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"--max-turns": {
							Description: "Example: Maximum API turns per session before stopping",
							Type:        "integer",
							Default:     float64(1000),
						},
					},
				},
				"settings": {
					Description: "Settings passed via --settings flag. Structure matches Claude CLI settings.json format. Any valid Claude CLI setting can be specified.",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"model": {
							Description: "Example: Claude model to use",
							Type:        "string",
							Default:     "claude-sonnet-4-5",
						},
						"permissions": {
							Description: "Example: Permission settings",
							Type:        "object",
						},
						"sandbox": {
							Description: "Example: Sandbox configuration",
							Type:        "object",
						},
					},
				},
			},
		},
	}
}
