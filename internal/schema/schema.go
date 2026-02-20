// Package schema generates documentation-oriented JSON schema from the
// agent harness configuration structs. The schema is hand-crafted to
// match the structure of HarnessConfig and its nested types.
package schema

import "github.com/cpplain/lorah/internal/config"

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
// The schema mirrors the Python schema.py generate_schema() output.
func GenerateSchema() Schema {
	return Schema{
		"model": {
			Description: "Claude model to use for agent execution",
			Type:        "string",
			Default:     config.DefaultModel,
			Options:     availableModels(),
		},
		"max_turns": {
			Description: "Maximum API turns per session before auto-continuing",
			Type:        "integer",
			Default:     float64(1000),
		},
		"max_iterations": {
			Description: "Maximum total sessions before stopping (null = unlimited)",
			Type:        "integer",
		},
		"auto_continue_delay": {
			Description: "Delay in seconds before auto-continuing to next session",
			Type:        "number",
			Default:     float64(3),
		},
		"tools": {
			Description: "Tool configuration",
			Type:        "object",
			Fields: map[string]FieldInfo{
				"builtin": {
					Description: "Built-in Claude SDK tools to enable",
					Type:        "array",
					Default:     defaultBuiltinTools(),
					Options:     config.DefaultBuiltinTools,
					Items:       &FieldInfo{Type: "string"},
				},
				"mcp_servers": {
					Description: "MCP (Model Context Protocol) servers to connect",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"command": {
							Description: "Command to launch the MCP server",
							Type:        "string",
						},
						"args": {
							Description: "Command-line arguments for the server",
							Type:        "array",
							Items:       &FieldInfo{Type: "string"},
						},
						"env": {
							Description: "Environment variables (supports ${VAR} expansion)",
							Type:        "object",
						},
					},
				},
			},
		},
		"security": {
			Description: "Security configuration",
			Type:        "object",
			Fields: map[string]FieldInfo{
				"permission_mode": {
					Description: "Permission mode controls how tool calls are approved",
					Type:        "string",
					Default:     string(config.PermissionModeAcceptEdits),
					Enum: []string{
						string(config.PermissionModeDefault),
						string(config.PermissionModeAcceptEdits),
						string(config.PermissionModeBypassPermissions),
						string(config.PermissionModePlan),
					},
				},
				"sandbox": {
					Description: "OS-level sandbox configuration",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"enabled": {
							Description: "Enable OS-level sandbox (strongly recommended)",
							Type:        "boolean",
							Default:     true,
						},
						"auto_allow_bash_if_sandboxed": {
							Description: "Auto-allow all Bash commands when sandbox is enabled",
							Type:        "boolean",
							Default:     true,
						},
						"allow_unsandboxed_commands": {
							Description: "Allow commands to run outside the sandbox (secure default: false)",
							Type:        "boolean",
							Default:     false,
						},
						"excluded_commands": {
							Description: "Commands to exclude from sandboxing (even when sandbox is enabled)",
							Type:        "array",
							Default:     []any{},
							Items:       &FieldInfo{Type: "string"},
						},
						"network": {
							Description: "Network access restrictions for sandboxed commands",
							Type:        "object",
							Fields: map[string]FieldInfo{
								"allowed_domains": {
									Description: "Domains the agent can access via network",
									Type:        "array",
									Default:     []any{},
									Items:       &FieldInfo{Type: "string"},
								},
								"allow_local_binding": {
									Description: "Allow binding to localhost addresses",
									Type:        "boolean",
									Default:     false,
								},
								"allow_unix_sockets": {
									Description: "Unix socket paths the agent can access",
									Type:        "array",
									Default:     []any{},
									Items:       &FieldInfo{Type: "string"},
								},
							},
						},
					},
				},
				"permissions": {
					Description: "Declarative allow/deny permission rules",
					Type:        "object",
					Fields: map[string]FieldInfo{
						"allow": {
							Description: "Patterns to explicitly allow (glob patterns)",
							Type:        "array",
							Default:     []any{},
							Items:       &FieldInfo{Type: "string"},
						},
						"deny": {
							Description: "Patterns to explicitly deny (takes precedence over allow)",
							Type:        "array",
							Default:     []any{},
							Items:       &FieldInfo{Type: "string"},
						},
					},
				},
			},
		},
		"error_recovery": {
			Description: "Error recovery and circuit breaker configuration",
			Type:        "object",
			Fields: map[string]FieldInfo{
				"max_consecutive_errors": {
					Description: "Maximum consecutive session errors before stopping",
					Type:        "integer",
					Default:     float64(5),
				},
				"initial_backoff_seconds": {
					Description: "Initial backoff delay after first error (seconds)",
					Type:        "number",
					Default:     float64(5),
				},
				"max_backoff_seconds": {
					Description: "Maximum backoff delay (capped exponential backoff)",
					Type:        "number",
					Default:     float64(120),
				},
				"backoff_multiplier": {
					Description: "Multiplier for exponential backoff",
					Type:        "number",
					Default:     float64(2),
				},
			},
		},
		"post_run_instructions": {
			Description: "Commands to display after agent completes",
			Type:        "array",
			Items:       &FieldInfo{Type: "string"},
		},
	}
}

// availableModels returns the list of available Claude models.
func availableModels() []string {
	return []string{
		"claude-opus-4-6",
		"claude-sonnet-4-6",
		"claude-haiku-4-5-20251001",
	}
}

// defaultBuiltinTools returns the default built-in tools as an []any for JSON serialization.
func defaultBuiltinTools() any {
	result := make([]any, len(config.DefaultBuiltinTools))
	for i, t := range config.DefaultBuiltinTools {
		result[i] = t
	}
	return result
}
