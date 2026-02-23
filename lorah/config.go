// Package config provides configuration loading, validation, and management
// for the agent harness. It reads .lorah/config.json and resolves
// file: references, environment variables, and CLI overrides.
package lorah

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigDirName is the name of the harness configuration directory.
const ConfigDirName = ".lorah"

// DefaultModel is the default Claude model.
const DefaultModel = "claude-sonnet-4-5"

// Fixed file names
const (
	// InitializationPromptFile is the fixed name of the initialization phase prompt.
	InitializationPromptFile = "prompts/initialization.md"
	// ImplementationPromptFile is the fixed name of the implementation phase prompt.
	ImplementationPromptFile = "prompts/implementation.md"
	// SpecFile is the fixed name of the project specification file.
	SpecFile = "spec.md"
)

// DefaultBuiltinTools are the default built-in tools enabled for the agent.
var DefaultBuiltinTools = []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}

// PermissionMode represents valid permission modes for tool call approval.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModePlan              PermissionMode = "plan"
)

// SandboxNetworkConfig holds configuration for sandbox network isolation.
type SandboxNetworkConfig struct {
	AllowedDomains    []string `json:"allowed_domains"`
	AllowLocalBinding bool     `json:"allow_local_binding"`
	AllowUnixSockets  []string `json:"allow_unix_sockets"`
}

// SandboxConfig holds configuration for OS-level sandbox.
type SandboxConfig struct {
	Enabled                  bool                 `json:"enabled"`
	AutoAllowBashIfSandboxed bool                 `json:"auto_allow_bash_if_sandboxed"`
	AllowUnsandboxedCommands bool                 `json:"allow_unsandboxed_commands"`
	ExcludedCommands         []string             `json:"excluded_commands"`
	Network                  SandboxNetworkConfig `json:"network"`
}

// PermissionRulesConfig holds declarative allow/deny permission rules.
type PermissionRulesConfig struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// SecurityConfig holds security configuration.
type SecurityConfig struct {
	PermissionMode string                `json:"permission_mode"`
	Sandbox        SandboxConfig         `json:"sandbox"`
	Permissions    PermissionRulesConfig `json:"permissions"`
}

// McpServerConfig holds configuration for an MCP server.
type McpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// ToolsConfig holds tools configuration.
type ToolsConfig struct {
	Builtin    []string                   `json:"builtin"`
	McpServers map[string]McpServerConfig `json:"mcp_servers"`
}

// ErrorRecoveryConfig holds configuration for error recovery behavior.
type ErrorRecoveryConfig struct {
	MaxConsecutiveErrors  int     `json:"max_consecutive_errors"`
	InitialBackoffSeconds float64 `json:"initial_backoff_seconds"`
	MaxBackoffSeconds     float64 `json:"max_backoff_seconds"`
	BackoffMultiplier     float64 `json:"backoff_multiplier"`
	MaxErrorMessageLength int     `json:"max_error_message_length"`
}

// HarnessConfig is the top-level configuration for the agent harness.
type HarnessConfig struct {
	// Agent
	Model string `json:"model"`

	// Session
	MaxTurns          int  `json:"max_turns"`
	MaxIterations     *int `json:"max_iterations"`
	AutoContinueDelay int  `json:"auto_continue_delay"`

	// Tools
	Tools ToolsConfig `json:"tools"`

	// Security
	Security SecurityConfig `json:"security"`

	// Error Recovery
	ErrorRecovery ErrorRecoveryConfig `json:"error_recovery"`

	// Post-run instructions
	PostRunInstructions []string `json:"post_run_instructions"`

	// Resolved paths (not from JSON)
	ProjectDir string `json:"-"`
	HarnessDir string `json:"-"`
}

// ConfigError is returned when configuration is invalid or cannot be loaded.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

func newConfigError(format string, args ...any) *ConfigError {
	return &ConfigError{Message: fmt.Sprintf(format, args...)}
}

// defaultConfig returns a HarnessConfig with all default values set.
func defaultConfig() HarnessConfig {
	return HarnessConfig{
		Model:             DefaultModel,
		MaxTurns:          1000,
		MaxIterations:     nil,
		AutoContinueDelay: 3,
		Tools: ToolsConfig{
			Builtin:    DefaultBuiltinTools,
			McpServers: map[string]McpServerConfig{},
		},
		Security: SecurityConfig{
			PermissionMode: string(PermissionModeAcceptEdits),
			Sandbox: SandboxConfig{
				Enabled:                  true,
				AutoAllowBashIfSandboxed: true,
				AllowUnsandboxedCommands: false,
				ExcludedCommands:         []string{},
				Network: SandboxNetworkConfig{
					AllowedDomains:    []string{},
					AllowLocalBinding: false,
					AllowUnixSockets:  []string{},
				},
			},
			Permissions: PermissionRulesConfig{
				Allow: []string{},
				Deny:  []string{},
			},
		},
		ErrorRecovery: ErrorRecoveryConfig{
			MaxConsecutiveErrors:  5,
			InitialBackoffSeconds: 5.0,
			MaxBackoffSeconds:     120.0,
			BackoffMultiplier:     2.0,
			MaxErrorMessageLength: 2000,
		},
		PostRunInstructions: []string{},
	}
}

// rawConfig is used for JSON unmarshalling with optional/nullable fields.
type rawConfig struct {
	Model               *string            `json:"model"`
	MaxTurns            *int               `json:"max_turns"`
	MaxIterations       *int               `json:"max_iterations"`
	AutoContinueDelay   *int               `json:"auto_continue_delay"`
	Tools               *rawToolsConfig    `json:"tools"`
	Security            *rawSecurityConfig `json:"security"`
	ErrorRecovery       *rawErrorConfig    `json:"error_recovery"`
	PostRunInstructions []string           `json:"post_run_instructions"`
}

type rawToolsConfig struct {
	Builtin    []string                      `json:"builtin"`
	McpServers map[string]rawMcpServerConfig `json:"mcp_servers"`
}

type rawMcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type rawSecurityConfig struct {
	PermissionMode *string               `json:"permission_mode"`
	Sandbox        *rawSandboxConfig     `json:"sandbox"`
	Permissions    *rawPermissionsConfig `json:"permissions"`
}

type rawSandboxConfig struct {
	Enabled                  *bool             `json:"enabled"`
	AutoAllowBashIfSandboxed *bool             `json:"auto_allow_bash_if_sandboxed"`
	AllowUnsandboxedCommands *bool             `json:"allow_unsandboxed_commands"`
	ExcludedCommands         []string          `json:"excluded_commands"`
	Network                  *rawNetworkConfig `json:"network"`
}

type rawNetworkConfig struct {
	AllowedDomains    []string `json:"allowed_domains"`
	AllowLocalBinding *bool    `json:"allow_local_binding"`
	AllowUnixSockets  []string `json:"allow_unix_sockets"`
}

type rawPermissionsConfig struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type rawErrorConfig struct {
	MaxConsecutiveErrors  *int     `json:"max_consecutive_errors"`
	InitialBackoffSeconds *float64 `json:"initial_backoff_seconds"`
	MaxBackoffSeconds     *float64 `json:"max_backoff_seconds"`
	BackoffMultiplier     *float64 `json:"backoff_multiplier"`
}

// mergeRawConfig merges raw JSON values over a default config.
func mergeRawConfig(cfg *HarnessConfig, raw *rawConfig) {
	if raw.Model != nil {
		cfg.Model = *raw.Model
	}
	if raw.MaxTurns != nil {
		cfg.MaxTurns = *raw.MaxTurns
	}
	if raw.MaxIterations != nil {
		v := *raw.MaxIterations
		cfg.MaxIterations = &v
	}
	if raw.AutoContinueDelay != nil {
		cfg.AutoContinueDelay = *raw.AutoContinueDelay
	}
	if raw.PostRunInstructions != nil {
		cfg.PostRunInstructions = raw.PostRunInstructions
	}

	if raw.Tools != nil {
		if raw.Tools.Builtin != nil {
			cfg.Tools.Builtin = raw.Tools.Builtin
		}
		if raw.Tools.McpServers != nil {
			cfg.Tools.McpServers = make(map[string]McpServerConfig)
			for name, srv := range raw.Tools.McpServers {
				args := srv.Args
				if args == nil {
					args = []string{}
				}
				envMap := make(map[string]string)
				for k, v := range srv.Env {
					expanded := os.ExpandEnv(v)
					envMap[k] = expanded
				}
				cfg.Tools.McpServers[name] = McpServerConfig{
					Command: srv.Command,
					Args:    args,
					Env:     envMap,
				}
			}
		}
	}

	if raw.Security != nil {
		s := raw.Security
		if s.PermissionMode != nil {
			cfg.Security.PermissionMode = *s.PermissionMode
		}
		if s.Sandbox != nil {
			sb := s.Sandbox
			if sb.Enabled != nil {
				cfg.Security.Sandbox.Enabled = *sb.Enabled
			}
			if sb.AutoAllowBashIfSandboxed != nil {
				cfg.Security.Sandbox.AutoAllowBashIfSandboxed = *sb.AutoAllowBashIfSandboxed
			}
			if sb.AllowUnsandboxedCommands != nil {
				cfg.Security.Sandbox.AllowUnsandboxedCommands = *sb.AllowUnsandboxedCommands
			}
			if sb.ExcludedCommands != nil {
				cfg.Security.Sandbox.ExcludedCommands = sb.ExcludedCommands
			}
			if sb.Network != nil {
				net := sb.Network
				if net.AllowedDomains != nil {
					cfg.Security.Sandbox.Network.AllowedDomains = net.AllowedDomains
				}
				if net.AllowLocalBinding != nil {
					cfg.Security.Sandbox.Network.AllowLocalBinding = *net.AllowLocalBinding
				}
				if net.AllowUnixSockets != nil {
					cfg.Security.Sandbox.Network.AllowUnixSockets = net.AllowUnixSockets
				}
			}
		}
		if s.Permissions != nil {
			p := s.Permissions
			if p.Allow != nil {
				cfg.Security.Permissions.Allow = p.Allow
			}
			if p.Deny != nil {
				cfg.Security.Permissions.Deny = p.Deny
			}
		}
	}

	if raw.ErrorRecovery != nil {
		er := raw.ErrorRecovery
		if er.MaxConsecutiveErrors != nil {
			cfg.ErrorRecovery.MaxConsecutiveErrors = *er.MaxConsecutiveErrors
		}
		if er.InitialBackoffSeconds != nil {
			cfg.ErrorRecovery.InitialBackoffSeconds = *er.InitialBackoffSeconds
		}
		if er.MaxBackoffSeconds != nil {
			cfg.ErrorRecovery.MaxBackoffSeconds = *er.MaxBackoffSeconds
		}
		if er.BackoffMultiplier != nil {
			cfg.ErrorRecovery.BackoffMultiplier = *er.BackoffMultiplier
		}
	}
}

// validateEnvExpansion checks that environment variable expansion in MCP server
// env values succeeded. It detects when ${VAR} syntax was used but the variable
// was undefined, which would cause os.ExpandEnv to silently return empty string.
func validateEnvExpansion(raw *rawConfig) []string {
	var errors []string
	if raw.Tools != nil && raw.Tools.McpServers != nil {
		for name, srv := range raw.Tools.McpServers {
			for k, v := range srv.Env {
				if strings.Contains(v, "${") {
					expanded := os.ExpandEnv(v)
					if expanded == "" {
						errors = append(errors, fmt.Sprintf(
							"tools.mcp_servers.%s.env.%s: environment variable expansion resulted in empty string (original: %q)",
							name, k, v,
						))
					}
				}
			}
		}
	}
	return errors
}

// validateConfig validates a HarnessConfig and returns a list of error messages.
func validateConfig(cfg *HarnessConfig) []string {
	var errors []string

	// Validate model
	if cfg.Model == "" {
		errors = append(errors, "model must be a non-empty string")
	}

	// Validate permission mode
	validModes := map[string]bool{
		string(PermissionModeDefault):           true,
		string(PermissionModeAcceptEdits):       true,
		string(PermissionModeBypassPermissions): true,
		string(PermissionModePlan):              true,
	}
	if !validModes[cfg.Security.PermissionMode] {
		errors = append(errors, fmt.Sprintf(
			"security.permission_mode must be one of [default acceptEdits bypassPermissions plan], got: %q",
			cfg.Security.PermissionMode,
		))
	}

	// Validate MCP server commands
	for name, server := range cfg.Tools.McpServers {
		if server.Command == "" {
			errors = append(errors, fmt.Sprintf(
				"tools.mcp_servers.%s.command must be a non-empty string", name,
			))
		}
	}

	// Validate max_turns
	if cfg.MaxTurns < 1 {
		errors = append(errors, "max_turns must be positive")
	}

	// Validate auto_continue_delay
	if cfg.AutoContinueDelay < 0 {
		errors = append(errors, "auto_continue_delay must be non-negative")
	}

	// Validate max_iterations
	if cfg.MaxIterations != nil && *cfg.MaxIterations <= 0 {
		errors = append(errors, "max_iterations must be positive when set")
	}

	// Validate error recovery
	if cfg.ErrorRecovery.MaxConsecutiveErrors < 1 {
		errors = append(errors, "error_recovery.max_consecutive_errors must be positive")
	}
	if cfg.ErrorRecovery.InitialBackoffSeconds <= 0 {
		errors = append(errors, "error_recovery.initial_backoff_seconds must be positive")
	}
	if cfg.ErrorRecovery.MaxBackoffSeconds < cfg.ErrorRecovery.InitialBackoffSeconds {
		errors = append(errors, "error_recovery.max_backoff_seconds must be >= initial_backoff_seconds")
	}
	if cfg.ErrorRecovery.BackoffMultiplier < 1.0 {
		errors = append(errors, "error_recovery.backoff_multiplier must be >= 1.0 (used for exponential backoff)")
	}

	return errors
}

// CLIOverrides holds CLI flag overrides that are applied after loading config.
type CLIOverrides struct {
	Model         string
	MaxIterations *int
}

// LoadConfig loads configuration from .lorah/config.json.
//
// It reads the config file, merges with defaults, expands env vars in MCP
// server env values, resolves file: references, applies CLI overrides, and
// validates the resulting config.
func LoadConfig(projectDir string, overrides *CLIOverrides) (*HarnessConfig, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, newConfigError("failed to resolve project dir: %v", err)
	}

	harnessDir := filepath.Join(absProjectDir, ConfigDirName)
	configFile := filepath.Join(harnessDir, "config.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, newConfigError("config file not found: %s", configFile)
		}
		return nil, newConfigError("failed to read config file %s: %v", configFile, err)
	}

	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, newConfigError("failed to parse %s: %v", configFile, err)
	}

	// Start with defaults
	cfg := defaultConfig()

	// Merge raw values over defaults
	mergeRawConfig(&cfg, &raw)

	// Validate environment variable expansion
	envErrs := validateEnvExpansion(&raw)
	if len(envErrs) > 0 {
		return nil, newConfigError("environment variable expansion errors:\n  %s", strings.Join(envErrs, "\n  "))
	}

	// Set resolved paths
	cfg.ProjectDir = absProjectDir
	cfg.HarnessDir = harnessDir

	// Apply CLI overrides
	if overrides != nil {
		if overrides.Model != "" {
			cfg.Model = overrides.Model
		}
		if overrides.MaxIterations != nil {
			v := *overrides.MaxIterations
			cfg.MaxIterations = &v
		}
	}

	// Validate
	errs := validateConfig(&cfg)
	if len(errs) > 0 {
		return nil, newConfigError("configuration errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return &cfg, nil
}
