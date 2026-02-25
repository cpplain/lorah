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

// Fixed file names
const (
	// ConfigFileName is the name of the harness config file.
	ConfigFileName = "config.json"
	// InitializationPromptFile is the fixed name of the initialization phase prompt.
	InitializationPromptFile = "prompts/initialization.md"
	// ImplementationPromptFile is the fixed name of the implementation phase prompt.
	ImplementationPromptFile = "prompts/implementation.md"
	// SpecFile is the fixed name of the project specification file.
	SpecFile = "spec.md"
)

// ErrorRecoveryConfig holds configuration for error recovery behavior.
type ErrorRecoveryConfig struct {
	MaxConsecutiveErrors  int     `json:"max-consecutive-errors"`
	InitialBackoffSeconds float64 `json:"initial-backoff-seconds"`
	MaxBackoffSeconds     float64 `json:"max-backoff-seconds"`
	BackoffMultiplier     float64 `json:"backoff-multiplier"`
	MaxErrorMessageLength int     `json:"max-error-message-length"`
}

// HarnessSettings holds harness-specific configuration for the agent loop.
type HarnessSettings struct {
	MaxIterations     *int                `json:"max-iterations"`
	AutoContinueDelay int                 `json:"auto-continue-delay"`
	ErrorRecovery     ErrorRecoveryConfig `json:"error-recovery"`
}

// ClaudeSection holds all Claude CLI configuration.
// Flags and settings are passthrough maps - lorah does not validate them.
// Claude CLI validates its own flags and settings.
type ClaudeSection struct {
	Flags    map[string]any `json:"flags"`
	Settings map[string]any `json:"settings"`
}

// HarnessConfig is the top-level configuration for the agent harness.
type HarnessConfig struct {
	// Harness-specific settings
	Harness HarnessSettings `json:"harness"`

	// Claude CLI configuration
	Claude ClaudeSection `json:"claude"`

	// Resolved paths (not from JSON)
	ProjectDir string `json:"-"`
	HarnessDir string `json:"-"`
}

// SettingsJSON serializes the Claude settings to JSON for the --settings flag.
func (cfg *HarnessConfig) SettingsJSON() (string, error) {
	data, err := json.Marshal(cfg.Claude.Settings)
	if err != nil {
		return "", fmt.Errorf("failed to serialize claude settings: %w", err)
	}
	return string(data), nil
}

// FormatConfig returns a human-readable JSON representation of the resolved config.
func FormatConfig(cfg *HarnessConfig) string {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return string(data)
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

// deepMerge recursively merges src into dst.
// Values in src override values in dst. Nested maps are merged recursively.
// Arrays and primitives are replaced, not merged.
func deepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for key, srcVal := range src {
		if srcMap, ok := srcVal.(map[string]any); ok {
			// If source value is a map, try to recursively merge
			if dstVal, exists := dst[key]; exists {
				if dstMap, ok := dstVal.(map[string]any); ok {
					dst[key] = deepMerge(dstMap, srcMap)
					continue
				}
			}
			// dst[key] doesn't exist or isn't a map - deep copy srcMap
			dst[key] = deepMerge(nil, srcMap)
		} else {
			// Not a map - replace directly
			dst[key] = srcVal
		}
	}
	return dst
}

// rawConfig is used for JSON unmarshalling with optional/nullable fields.
type rawConfig struct {
	Harness *rawHarnessConfig `json:"harness"`
	Claude  map[string]any    `json:"claude"`
}

type rawHarnessConfig struct {
	MaxIterations     *int            `json:"max-iterations"`
	AutoContinueDelay *int            `json:"auto-continue-delay"`
	ErrorRecovery     *rawErrorConfig `json:"error-recovery"`
}

type rawErrorConfig struct {
	MaxConsecutiveErrors  *int     `json:"max-consecutive-errors"`
	InitialBackoffSeconds *float64 `json:"initial-backoff-seconds"`
	MaxBackoffSeconds     *float64 `json:"max-backoff-seconds"`
	BackoffMultiplier     *float64 `json:"backoff-multiplier"`
	MaxErrorMessageLength *int     `json:"max-error-message-length"`
}

// mergeRawHarnessConfig merges raw JSON values for the harness section only.
// Claude section is handled separately via deepMerge.
func mergeRawHarnessConfig(cfg *HarnessSettings, raw *rawHarnessConfig) {
	if raw == nil {
		return
	}
	if raw.MaxIterations != nil {
		v := *raw.MaxIterations
		cfg.MaxIterations = &v
	}
	if raw.AutoContinueDelay != nil {
		cfg.AutoContinueDelay = *raw.AutoContinueDelay
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
		if er.MaxErrorMessageLength != nil {
			cfg.ErrorRecovery.MaxErrorMessageLength = *er.MaxErrorMessageLength
		}
	}
}

// validateConfig validates a HarnessConfig and returns a list of error messages.
func validateConfig(cfg *HarnessConfig) []string {
	var errors []string

	// Validate harness.auto-continue-delay
	if cfg.Harness.AutoContinueDelay < 0 {
		errors = append(errors, "harness.auto-continue-delay must be non-negative")
	}

	// Validate harness.max-iterations
	if cfg.Harness.MaxIterations != nil && *cfg.Harness.MaxIterations <= 0 {
		errors = append(errors, "harness.max-iterations must be positive when set")
	}

	// Validate harness.error-recovery
	if cfg.Harness.ErrorRecovery.MaxConsecutiveErrors < 1 {
		errors = append(errors, "harness.error-recovery.max-consecutive-errors must be positive")
	}
	if cfg.Harness.ErrorRecovery.InitialBackoffSeconds <= 0 {
		errors = append(errors, "harness.error-recovery.initial-backoff-seconds must be positive")
	}
	if cfg.Harness.ErrorRecovery.MaxBackoffSeconds < cfg.Harness.ErrorRecovery.InitialBackoffSeconds {
		errors = append(errors, "harness.error-recovery.max-backoff-seconds must be >= initial-backoff-seconds")
	}
	if cfg.Harness.ErrorRecovery.BackoffMultiplier < 1.0 {
		errors = append(errors, "harness.error-recovery.backoff-multiplier must be >= 1.0 (used for exponential backoff)")
	}

	// Claude config is not validated - Claude CLI handles its own validation

	return errors
}

// CLIOverrides holds CLI flag overrides that are applied after loading config.
type CLIOverrides struct {
	MaxIterations *int
}

// LoadConfig loads configuration from .lorah/config.json.
//
// It loads defaults from the embedded template, merges user config over defaults,
// applies CLI overrides, and validates the resulting config.
func LoadConfig(projectDir string, overrides *CLIOverrides) (*HarnessConfig, error) {
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, newConfigError("failed to resolve project dir: %v", err)
	}

	harnessDir := filepath.Join(absProjectDir, ConfigDirName)
	configFile := filepath.Join(harnessDir, ConfigFileName)

	// Load defaults from embedded template
	var templateRaw struct {
		Harness *rawHarnessConfig `json:"harness"`
		Claude  map[string]any    `json:"claude"`
	}
	if err := json.Unmarshal(templateConfigJSON, &templateRaw); err != nil {
		return nil, newConfigError("failed to parse embedded config template: %v", err)
	}

	// Initialize config with base harness defaults
	cfg := HarnessConfig{
		Harness: HarnessSettings{
			MaxIterations:     nil,
			AutoContinueDelay: 3,
			ErrorRecovery: ErrorRecoveryConfig{
				MaxConsecutiveErrors:  5,
				InitialBackoffSeconds: 5.0,
				MaxBackoffSeconds:     120.0,
				BackoffMultiplier:     2.0,
				MaxErrorMessageLength: 2000,
			},
		},
		Claude: ClaudeSection{
			Flags:    make(map[string]any),
			Settings: make(map[string]any),
		},
	}

	// Merge harness section from template
	mergeRawHarnessConfig(&cfg.Harness, templateRaw.Harness)

	// Copy claude section from template
	if templateRaw.Claude != nil {
		if flags, ok := templateRaw.Claude["flags"].(map[string]any); ok {
			cfg.Claude.Flags = deepMerge(cfg.Claude.Flags, flags)
		}
		if settings, ok := templateRaw.Claude["settings"].(map[string]any); ok {
			cfg.Claude.Settings = deepMerge(cfg.Claude.Settings, settings)
		}
	}

	// Load and merge user config if present
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, newConfigError("failed to read config file %s: %v", configFile, err)
		}
		// File doesn't exist - use template defaults
	} else {
		// File exists - parse and merge over template defaults
		var raw rawConfig
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, newConfigError("failed to parse %s: %v", configFile, err)
		}

		// Merge harness section (typed)
		mergeRawHarnessConfig(&cfg.Harness, raw.Harness)

		// Merge claude section (deep merge maps)
		if raw.Claude != nil {
			if flags, ok := raw.Claude["flags"].(map[string]any); ok {
				cfg.Claude.Flags = deepMerge(cfg.Claude.Flags, flags)
			}
			if settings, ok := raw.Claude["settings"].(map[string]any); ok {
				cfg.Claude.Settings = deepMerge(cfg.Claude.Settings, settings)
			}
		}
	}

	// Set resolved paths
	cfg.ProjectDir = absProjectDir
	cfg.HarnessDir = harnessDir

	// Apply CLI overrides
	if overrides != nil {
		if overrides.MaxIterations != nil {
			v := *overrides.MaxIterations
			cfg.Harness.MaxIterations = &v
		}
	}

	// Validate harness section only
	errs := validateConfig(&cfg)
	if len(errs) > 0 {
		return nil, newConfigError("configuration errors:\n  %s", strings.Join(errs, "\n  "))
	}

	return &cfg, nil
}
