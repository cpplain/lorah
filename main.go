// Command lorah is the CLI entry point for the agent harness.
// It provides subcommands for running, verifying, initializing, and
// getting information about harness configurations.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cpplain/lorah/lorah"
)

// Version is the current version of lorah.
// Set at build time via: -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	// Top-level flags
	fs := flag.NewFlagSet("lorah", flag.ContinueOnError)
	versionFlag := fs.Bool("version", false, "Print version and exit")
	vFlag := fs.Bool("V", false, "Print version and exit")

	// Custom usage
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah <command> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Generic harness for long-running autonomous coding agents\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  run     Run the agent loop\n")
		fmt.Fprintf(os.Stderr, "  verify  Check setup and configuration\n")
		fmt.Fprintf(os.Stderr, "  init    Scaffold .lorah/ with starter config\n")
		fmt.Fprintf(os.Stderr, "  info    Get templates, schema, and documentation\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	// Parse only top-level flags, stop at first non-flag argument (the subcommand)
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if *versionFlag || *vFlag {
		fmt.Printf("lorah %s\n", Version)
		os.Exit(0)
	}

	args := fs.Args()
	if len(args) == 0 {
		fs.Usage()
		os.Exit(0)
	}

	switch args[0] {
	case "run":
		cmdRun(args[1:])
	case "verify":
		cmdVerify(args[1:])
	case "init":
		cmdInit(args[1:])
	case "info":
		cmdInfo(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", args[0])
		fs.Usage()
		os.Exit(1)
	}
}

// resolveProjectDir resolves the project directory to an absolute path.
// If projectDir is empty, uses the current working directory.
func resolveProjectDir(projectDir string) (string, error) {
	if projectDir == "" {
		return os.Getwd()
	}
	return filepath.Abs(projectDir)
}

// addProjectDirFlag adds --project-dir to a FlagSet and returns a pointer to
// the flag value.
func addProjectDirFlag(fs *flag.FlagSet) *string {
	return fs.String("project-dir", "", "Agent's working directory (default: current directory)")
}

// cmdRun implements the 'run' subcommand.
func cmdRun(args []string) {
	fs := flag.NewFlagSet("lorah run", flag.ExitOnError)
	projectDir := addProjectDirFlag(fs)
	model := fs.String("model", "", "Override model from config")
	maxIterations := fs.Int("max-iterations", 0, "Override max iterations from config")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah run [options]\n\n")
		fmt.Fprintf(os.Stderr, "Run the agent loop\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	dir, err := resolveProjectDir(*projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving project directory: %v\n", err)
		os.Exit(1)
	}

	// Build CLI overrides from flags
	overrides := &lorah.CLIOverrides{}
	if *model != "" {
		overrides.Model = *model
	}
	if *maxIterations != 0 {
		overrides.MaxIterations = maxIterations
	}

	cfg, err := lorah.LoadConfig(dir, overrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := lorah.RunAgent(ctx, cfg); err != nil {
		if errors.Is(err, lorah.ErrInterrupted) {
			fmt.Println("\n\nInterrupted by user")
			fmt.Println("To resume, run the same command again")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "\nFatal error: %v\n", err)
		os.Exit(1)
	}
}

// cmdVerify implements the 'verify' subcommand.
func cmdVerify(args []string) {
	fs := flag.NewFlagSet("lorah verify", flag.ExitOnError)
	projectDir := addProjectDirFlag(fs)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah verify [options]\n\n")
		fmt.Fprintf(os.Stderr, "Check setup and configuration\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	dir, err := resolveProjectDir(*projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving project directory: %v\n", err)
		os.Exit(1)
	}

	results := lorah.RunVerify(dir)

	fmt.Println("\nVerification Results:")
	fmt.Println("--------------------------------------------------")

	for _, result := range results {
		fmt.Println(result.String())
	}

	fmt.Println("--------------------------------------------------")

	var fails, warns, passes int
	for _, r := range results {
		switch r.Status {
		case "FAIL":
			fails++
		case "WARN":
			warns++
		case "PASS":
			passes++
		}
	}

	fmt.Printf("\n  %d passed, %d warnings, %d failed\n", passes, warns, fails)

	if fails > 0 {
		fmt.Println("\n  Fix the FAIL items above before running the agent.")
		os.Exit(1)
	}

	if warns > 0 {
		fmt.Println("\n  Warnings are non-blocking but may cause issues.")
	}
}

// cmdInit implements the 'init' subcommand.
func cmdInit(args []string) {
	fs := flag.NewFlagSet("lorah init", flag.ExitOnError)
	projectDir := addProjectDirFlag(fs)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah init [options]\n\n")
		fmt.Fprintf(os.Stderr, "Scaffold .lorah/ with starter config\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	dir, err := resolveProjectDir(*projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving project directory: %v\n", err)
		os.Exit(1)
	}

	if err := lorah.InitProject(dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdInfo implements the 'info' subcommand.
func cmdInfo(args []string) {
	fs := flag.NewFlagSet("lorah info", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah info <topic> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Get templates, schema, and documentation\n\n")
		fmt.Fprintf(os.Stderr, "Topics:\n")
		fmt.Fprintf(os.Stderr, "  template  Get template files\n")
		fmt.Fprintf(os.Stderr, "  schema    Get configuration schema\n")
		fmt.Fprintf(os.Stderr, "  preset    Get preset configurations\n")
		fmt.Fprintf(os.Stderr, "  guide     Get setup guide\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintf(os.Stderr, "Error: info subcommand required\n")
		fmt.Fprintf(os.Stderr, "Usage: lorah info {template|schema|preset|guide} [options]\n")
		os.Exit(1)
	}

	switch remaining[0] {
	case "template":
		cmdInfoTemplate(remaining[1:])
	case "schema":
		cmdInfoSchema(remaining[1:])
	case "preset":
		cmdInfoPreset(remaining[1:])
	case "guide":
		cmdInfoGuide(remaining[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown info topic %q\n", remaining[0])
		fmt.Fprintf(os.Stderr, "Usage: lorah info {template|schema|preset|guide} [options]\n")
		os.Exit(1)
	}
}

// cmdInfoTemplate implements 'lorah info template'.
func cmdInfoTemplate(args []string) {
	fs := flag.NewFlagSet("lorah info template", flag.ExitOnError)
	name := fs.String("name", "", "Template name (e.g. lorah.json)")
	listFlag := fs.Bool("list", false, "List all templates")
	allFlag := fs.Bool("all", false, "Get all templates with content")
	jsonFlag := fs.Bool("json", false, "Output as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah info template [options]\n\n")
		fmt.Fprintf(os.Stderr, "Get template files\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := lorah.CmdInfoTemplate(*name, *listFlag, *allFlag, *jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdInfoSchema implements 'lorah info schema'.
func cmdInfoSchema(args []string) {
	fs := flag.NewFlagSet("lorah info schema", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah info schema [options]\n\n")
		fmt.Fprintf(os.Stderr, "Get configuration schema\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := lorah.CmdInfoSchema(*jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdInfoPreset implements 'lorah info preset'.
func cmdInfoPreset(args []string) {
	fs := flag.NewFlagSet("lorah info preset", flag.ExitOnError)
	name := fs.String("name", "", "Preset name (e.g. python)")
	listFlag := fs.Bool("list", false, "List all presets")
	jsonFlag := fs.Bool("json", false, "Output as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah info preset [options]\n\n")
		fmt.Fprintf(os.Stderr, "Get preset configurations\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := lorah.CmdInfoPreset(*name, *listFlag, *jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// cmdInfoGuide implements 'lorah info guide'.
func cmdInfoGuide(args []string) {
	fs := flag.NewFlagSet("lorah info guide", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output as JSON")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lorah info guide [options]\n\n")
		fmt.Fprintf(os.Stderr, "Get setup guide\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if err := lorah.CmdInfoGuide(*jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
