// Package lorah provides setup verification checks for the agent harness.
// It checks that the environment, authentication, claude CLI, and configuration
// are all properly set up before running the agent.
package lorah

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// versionPattern extracts major.minor.patch from CLI version output.
var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Minimum Go version required by the project.
// IMPORTANT: This must match the version specified in go.mod.
const (
	minGoMajor   = 1
	minGoMinor   = 25
	minGoVersion = "1.25.0"
)

// CheckResult holds the result of a single verification check.
type CheckResult struct {
	Name    string
	Status  string // "PASS", "WARN", or "FAIL"
	Message string
}

// String formats the check result for display.
//
// Format: "  [STATUS]  Name - Message"
// The status field is padded to 8 characters total (including brackets).
func (r CheckResult) String() string {
	status := fmt.Sprintf("[%s]", r.Status)
	line := fmt.Sprintf("  %-8s %s", status, r.Name)
	if r.Message != "" {
		line += " - " + r.Message
	}
	return line
}

// CheckGoVersion verifies that the Go runtime version meets requirements.
// The minimum required version is defined by minGoVersion and must match go.mod.
// Returns WARN for versions below minimum to allow the harness to proceed.
func CheckGoVersion() CheckResult {
	version := runtime.Version()
	// runtime.Version() returns "go1.21.0" format
	versionStr := strings.TrimPrefix(version, "go")

	// Parse version to validate minimum requirement
	matches := versionPattern.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		// Cannot parse version; return PASS with the raw version string
		return CheckResult{Name: "Go version", Status: "PASS", Message: versionStr}
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	// Check against minimum version (must match go.mod)
	if major < minGoMajor || (major == minGoMajor && minor < minGoMinor) {
		return CheckResult{
			Name:    "Go version",
			Status:  "WARN",
			Message: fmt.Sprintf("%s is below minimum %s", matches[0], minGoVersion),
		}
	}

	return CheckResult{Name: "Go version", Status: "PASS", Message: matches[0]}
}

// CheckAPIConnectivity verifies that the claude CLI can reach the API and
// authenticate successfully by running a minimal one-turn prompt.
func CheckAPIConnectivity() CheckResult {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return CheckResult{
			Name:    "API connectivity",
			Status:  "FAIL",
			Message: "claude not found in PATH",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, claudePath, "-p", "hi", "--max-turns", "1", "--output-format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return CheckResult{Name: "API connectivity", Status: "FAIL", Message: msg}
	}
	return CheckResult{Name: "API connectivity", Status: "PASS"}
}

// CheckClaudeCLI verifies that the claude CLI is available and meets the minimum
// version requirement. It uses exec.LookPath to find the binary in the user's
// PATH. If the major version is below 2, a WARN result is returned rather than
// a FAIL so the harness can still proceed. On successful version parse, the
// version string is appended to the PASS message.
func CheckClaudeCLI() CheckResult {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return CheckResult{
			Name:    "Claude CLI",
			Status:  "FAIL",
			Message: "not found in PATH; install with: npm install -g @anthropic-ai/claude-code",
		}
	}

	cmd := exec.Command(claudePath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Name:    "Claude CLI",
			Status:  "WARN",
			Message: "Found but could not get version",
		}
	}

	raw := strings.TrimSpace(string(out))
	matches := versionPattern.FindStringSubmatch(raw)
	if len(matches) < 4 {
		// Cannot parse version; return WARN since we can't validate requirements.
		return CheckResult{Name: "Claude CLI", Status: "WARN", Message: fmt.Sprintf("Found but could not parse version: %s", raw)}
	}

	version := matches[0]
	major, _ := strconv.Atoi(matches[1])

	if major < 2 {
		return CheckResult{
			Name:    "Claude CLI",
			Status:  "WARN",
			Message: fmt.Sprintf("version %s is below minimum 2.0.0; upgrade with: npm install -g @anthropic-ai/claude-code", version),
		}
	}

	return CheckResult{Name: "Claude CLI", Status: "PASS", Message: version}
}

// CheckConfigExists verifies that the config.json file exists in the harness directory.
func CheckConfigExists(harnessDir string) CheckResult {
	configFile := filepath.Join(harnessDir, "config.json")
	if _, err := os.Stat(configFile); err == nil {
		return CheckResult{Name: "Config file", Status: "PASS", Message: configFile}
	}
	return CheckResult{
		Name:    "Config file",
		Status:  "FAIL",
		Message: fmt.Sprintf("Not found: %s", configFile),
	}
}

// CheckConfigValid attempts to load and validate the config file.
// It returns both the check result and the loaded config (nil on failure).
func CheckConfigValid(projectDir string) (CheckResult, *HarnessConfig) {
	cfg, err := LoadConfig(projectDir, nil)
	if err != nil {
		return CheckResult{
			Name:    "Config validation",
			Status:  "FAIL",
			Message: err.Error(),
		}, nil
	}
	return CheckResult{Name: "Config validation", Status: "PASS"}, cfg
}

// CheckRequiredFiles verifies that required files exist in the harness directory.
func CheckRequiredFiles(harnessDir string) CheckResult {
	requiredFiles := []struct {
		path string
		name string
	}{
		{filepath.Join(harnessDir, TaskListFile), TaskListFile},
		{filepath.Join(harnessDir, AgentProgressFile), AgentProgressFile},
		{filepath.Join(harnessDir, "prompts", "initialization.md"), "prompts/initialization.md"},
		{filepath.Join(harnessDir, "prompts", "implementation.md"), "prompts/implementation.md"},
	}

	var missing []string
	for _, f := range requiredFiles {
		if _, err := os.Stat(f.path); os.IsNotExist(err) {
			missing = append(missing, f.name)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:    "Required files",
			Status:  "FAIL",
			Message: fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")),
		}
	}

	return CheckResult{Name: "Required files", Status: "PASS"}
}

// CheckMCPCommands verifies that all MCP server commands are available on PATH.
// npx commands trigger a warning (auto-download) rather than a failure.
func CheckMCPCommands(cfg *HarnessConfig) CheckResult {
	if len(cfg.Tools.McpServers) == 0 {
		return CheckResult{Name: "MCP servers", Status: "PASS", Message: "None configured"}
	}

	var missing []string
	missingNPX := false

	for name, server := range cfg.Tools.McpServers {
		if _, err := exec.LookPath(server.Command); err != nil {
			if server.Command == "npx" {
				missingNPX = true
			} else {
				missing = append(missing, fmt.Sprintf("%s (%s)", name, server.Command))
			}
		}
	}

	// If only npx is missing, warn with a specific message
	if missingNPX && len(missing) == 0 {
		return CheckResult{
			Name:    "MCP servers",
			Status:  "WARN",
			Message: "npx not found on PATH (packages will auto-download on first run)",
		}
	}

	// If other commands are missing, report them
	if len(missing) > 0 {
		return CheckResult{
			Name:    "MCP servers",
			Status:  "WARN",
			Message: fmt.Sprintf("Commands not found: %s", strings.Join(missing, ", ")),
		}
	}

	// All commands found — list the server names
	names := make([]string, 0, len(cfg.Tools.McpServers))
	for name := range cfg.Tools.McpServers {
		names = append(names, name)
	}
	return CheckResult{
		Name:    "MCP servers",
		Status:  "PASS",
		Message: strings.Join(names, ", "),
	}
}

// CheckProjectDir verifies that the project directory is writable.
// If it doesn't exist, checks that the parent directory is writable.
func CheckProjectDir(projectDir string) CheckResult {
	info, err := os.Stat(projectDir)
	if err == nil && info.IsDir() {
		if isDirWritable(projectDir) {
			return CheckResult{Name: "Project directory", Status: "PASS", Message: projectDir}
		}
		return CheckResult{
			Name:    "Project directory",
			Status:  "FAIL",
			Message: fmt.Sprintf("Not writable: %s", projectDir),
		}
	}

	// Directory doesn't exist — check parent
	parent := filepath.Dir(projectDir)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		if isDirWritable(parent) {
			return CheckResult{
				Name:    "Project directory",
				Status:  "PASS",
				Message: fmt.Sprintf("Will be created: %s", projectDir),
			}
		}
	}
	return CheckResult{
		Name:    "Project directory",
		Status:  "FAIL",
		Message: fmt.Sprintf("Parent not writable: %s", parent),
	}
}

// isDirWritable checks if the given directory is writable by attempting to
// create a temporary file in it.
func isDirWritable(dir string) bool {
	tmp, err := os.CreateTemp(dir, ".writable-check-*")
	if err != nil {
		return false
	}
	tmp.Close()
	os.Remove(tmp.Name())
	return true
}

// RunVerify runs all verification checks and returns the results.
// All checks run regardless of individual failures.
func RunVerify(projectDir string) []CheckResult {
	harnessDir := filepath.Join(projectDir, ConfigDirName)

	var results []CheckResult

	// Independent environment checks
	results = append(results, CheckGoVersion())
	results = append(results, CheckClaudeCLI())
	results = append(results, CheckAPIConnectivity())

	// Config existence and validation
	results = append(results, CheckConfigExists(harnessDir))

	configResult, cfg := CheckConfigValid(projectDir)
	results = append(results, configResult)

	// Config-dependent checks (only if config loaded successfully)
	if cfg != nil {
		results = append(results, CheckRequiredFiles(harnessDir))
		results = append(results, CheckMCPCommands(cfg))
		results = append(results, CheckProjectDir(projectDir))
	}

	return results
}
