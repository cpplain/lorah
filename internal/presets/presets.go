// Package presets provides embedded preset configurations for common use cases.
// Each preset provides recommended settings for specific project types.
package presets

// Preset holds a named configuration preset.
type Preset struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config"`
}

// presets is the registry of all built-in presets.
var presets = map[string]Preset{
	"python": {
		Name:        "python",
		Description: "Python development with pip/uv and PyPI access",
		Config: map[string]any{
			"security.sandbox.network.allowed_domains": []string{
				"pypi.org",
				"files.pythonhosted.org",
				"github.com",
			},
			"security.permissions.allow": []string{
				"Bash(python *)",
				"Bash(pip *)",
				"Bash(uv *)",
				"Bash(git *)",
			},
		},
	},
	"go": {
		Name:        "go",
		Description: "Go development with module proxy access",
		Config: map[string]any{
			"security.sandbox.network.allowed_domains": []string{
				"proxy.golang.org",
				"sum.golang.org",
				"storage.googleapis.com",
				"github.com",
			},
			"security.permissions.allow": []string{
				"Bash(go *)",
				"Bash(git *)",
			},
		},
	},
	"rust": {
		Name:        "rust",
		Description: "Rust development with crates.io access",
		Config: map[string]any{
			"security.sandbox.network.allowed_domains": []string{
				"crates.io",
				"static.crates.io",
				"github.com",
			},
			"security.permissions.allow": []string{
				"Bash(cargo *)",
				"Bash(rustc *)",
				"Bash(git *)",
			},
		},
	},
	"web-nodejs": {
		Name:        "web-nodejs",
		Description: "Node.js/npm with registry access and local dev server binding",
		Config: map[string]any{
			"security.sandbox.network.allowed_domains": []string{
				"registry.npmjs.org",
				"github.com",
				"cdn.jsdelivr.net",
			},
			"security.sandbox.network.allow_local_binding": true,
			"security.permissions.allow": []string{
				"Bash(npm *)",
				"Bash(node *)",
				"Bash(npx *)",
				"Bash(git *)",
			},
		},
	},
	"read-only": {
		Name:        "read-only",
		Description: "Code analysis only - no write tools or network access",
		Config: map[string]any{
			"tools.builtin":                            []string{"Read", "Glob", "Grep"},
			"security.permission_mode":                 "bypassPermissions",
			"security.sandbox.network.allowed_domains": []string{},
		},
	},
}

// presetOrder defines the canonical order for listing presets.
var presetOrder = []string{"python", "go", "rust", "web-nodejs", "read-only"}

// GetPreset returns the preset with the given name, or nil if not found.
func GetPreset(name string) *Preset {
	p, ok := presets[name]
	if !ok {
		return nil
	}
	return &p
}

// ListPresets returns summary info for all presets in canonical order.
func ListPresets() []PresetSummary {
	summaries := make([]PresetSummary, 0, len(presetOrder))
	for _, name := range presetOrder {
		p, ok := presets[name]
		if ok {
			summaries = append(summaries, PresetSummary{
				Name:        p.Name,
				Description: p.Description,
			})
		}
	}
	return summaries
}

// PresetSummary holds just name and description for listing.
type PresetSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
