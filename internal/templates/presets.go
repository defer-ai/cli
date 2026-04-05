package templates

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Preset is a collection of pre-defined decisions for a common project type.
type Preset struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Decisions   []PresetDecision `yaml:"decisions"`
	Source      string           `yaml:"-"` // "builtin", "project", "global"
}

// PresetDecision is a single decision within a preset.
type PresetDecision struct {
	Category  string         `yaml:"category"`
	Question  string         `yaml:"question"`
	Options   []PresetOption `yaml:"options"`
	Impact    int            `yaml:"impact"`
	DependsOn []string       `yaml:"dependsOn,omitempty"`
	Context   string         `yaml:"context,omitempty"`
}

// PresetOption is one possible answer within a preset decision.
type PresetOption struct {
	Key   string `yaml:"key"`
	Label string `yaml:"label"`
}

// LoadPresetFile loads a single YAML preset file from disk.
func LoadPresetFile(path string) (*Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Preset
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		return nil, &PresetError{Path: path, Msg: "preset missing required 'name' field"}
	}
	return &p, nil
}

// PresetError reports a problem with a preset file.
type PresetError struct {
	Path string
	Msg  string
}

func (e *PresetError) Error() string {
	return e.Path + ": " + e.Msg
}

// DiscoverPresets discovers presets from three sources (later overrides earlier by name):
//  1. Built-in presets
//  2. Global presets: ~/.defer/templates/*.yaml
//  3. Project-local presets: <cwd>/.defer/templates/*.yaml
func DiscoverPresets(cwd string) []Preset {
	byName := make(map[string]Preset)

	// 1. Built-in
	for _, p := range DefaultPresets() {
		byName[p.Name] = p
	}

	// 2. Global (~/.defer/templates/*.yaml)
	if home, err := os.UserHomeDir(); err == nil {
		loadPresetsFromDir(filepath.Join(home, ".defer", "templates"), "global", byName)
	}

	// 3. Project-local (<cwd>/.defer/templates/*.yaml)
	loadPresetsFromDir(filepath.Join(cwd, ".defer", "templates"), "project", byName)

	// Collect in deterministic order: builtins first (sorted), then others alphabetically
	var result []Preset
	seen := make(map[string]bool)
	// Add builtins in their canonical order
	for _, bp := range DefaultPresets() {
		if p, ok := byName[bp.Name]; ok {
			result = append(result, p)
			seen[bp.Name] = true
		}
	}
	// Add any remaining (global/project-only) sorted by name
	var extra []string
	for name := range byName {
		if !seen[name] {
			extra = append(extra, name)
		}
	}
	sortStrings(extra)
	for _, name := range extra {
		result = append(result, byName[name])
	}
	return result
}

// sortStrings sorts a slice of strings in place (simple insertion sort to avoid importing sort).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// loadPresetsFromDir reads all *.yaml files from dir and adds them to byName.
func loadPresetsFromDir(dir, source string, byName map[string]Preset) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		p, err := LoadPresetFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		p.Source = source
		byName[p.Name] = *p
	}
}

// DefaultPresets returns the three built-in presets.
func DefaultPresets() []Preset {
	return []Preset{
		restAPIPreset(),
		cliToolPreset(),
		webAppPreset(),
	}
}

func restAPIPreset() Preset {
	return Preset{
		Name:        "rest-api",
		Description: "Common decisions for REST API projects",
		Source:      "builtin",
		Decisions: []PresetDecision{
			{
				Category: "Stack",
				Question: "Backend framework?",
				Options: []PresetOption{
					{Key: "A", Label: "Express"},
					{Key: "B", Label: "FastAPI"},
					{Key: "C", Label: "Gin"},
					{Key: "D", Label: "Spring Boot"},
				},
				Impact:  9,
				Context: "Foundational choice that determines language and ecosystem",
			},
			{
				Category: "Stack",
				Question: "Primary language?",
				Options: []PresetOption{
					{Key: "A", Label: "TypeScript"},
					{Key: "B", Label: "Python"},
					{Key: "C", Label: "Go"},
					{Key: "D", Label: "Java"},
				},
				Impact:    10,
				DependsOn: []string{},
				Context:   "Must align with chosen framework",
			},
			{
				Category: "Data",
				Question: "Primary database?",
				Options: []PresetOption{
					{Key: "A", Label: "PostgreSQL"},
					{Key: "B", Label: "MySQL"},
					{Key: "C", Label: "MongoDB"},
					{Key: "D", Label: "SQLite"},
				},
				Impact:  8,
				Context: "Affects data modeling and query patterns",
			},
			{
				Category: "Data",
				Question: "ORM / query builder?",
				Options: []PresetOption{
					{Key: "A", Label: "Prisma"},
					{Key: "B", Label: "SQLAlchemy"},
					{Key: "C", Label: "GORM"},
					{Key: "D", Label: "Raw SQL"},
				},
				Impact:  6,
				Context: "Depends on language and database choice",
			},
			{
				Category: "API",
				Question: "API style?",
				Options: []PresetOption{
					{Key: "A", Label: "REST (JSON)"},
					{Key: "B", Label: "GraphQL"},
					{Key: "C", Label: "gRPC"},
				},
				Impact:  8,
				Context: "Determines client-server contract format",
			},
			{
				Category: "API",
				Question: "Authentication method?",
				Options: []PresetOption{
					{Key: "A", Label: "JWT"},
					{Key: "B", Label: "Session cookies"},
					{Key: "C", Label: "OAuth2 / OIDC"},
					{Key: "D", Label: "API keys"},
				},
				Impact:  7,
				Context: "Security-critical; affects every endpoint",
			},
			{
				Category: "Deploy",
				Question: "Hosting platform?",
				Options: []PresetOption{
					{Key: "A", Label: "AWS"},
					{Key: "B", Label: "GCP"},
					{Key: "C", Label: "Fly.io"},
					{Key: "D", Label: "Self-hosted"},
				},
				Impact:  7,
				Context: "Affects CI/CD, infrastructure-as-code, and cost",
			},
		},
	}
}

func cliToolPreset() Preset {
	return Preset{
		Name:        "cli-tool",
		Description: "Common decisions for CLI tool projects",
		Source:      "builtin",
		Decisions: []PresetDecision{
			{
				Category: "Stack",
				Question: "Implementation language?",
				Options: []PresetOption{
					{Key: "A", Label: "Go"},
					{Key: "B", Label: "Rust"},
					{Key: "C", Label: "Python"},
					{Key: "D", Label: "TypeScript (Node)"},
				},
				Impact:  10,
				Context: "Determines distribution model and runtime requirements",
			},
			{
				Category: "UI",
				Question: "TUI framework?",
				Options: []PresetOption{
					{Key: "A", Label: "Bubble Tea"},
					{Key: "B", Label: "Ratatui"},
					{Key: "C", Label: "Rich / Textual"},
					{Key: "D", Label: "Ink"},
					{Key: "E", Label: "None (simple stdout)"},
				},
				Impact:  7,
				Context: "Defines how interactive the CLI experience is",
			},
			{
				Category: "UI",
				Question: "Output format?",
				Options: []PresetOption{
					{Key: "A", Label: "Plain text"},
					{Key: "B", Label: "JSON"},
					{Key: "C", Label: "Table"},
					{Key: "D", Label: "Multiple (flag-selectable)"},
				},
				Impact:  5,
				Context: "Affects scriptability and human readability",
			},
			{
				Category: "Distribution",
				Question: "Package manager / distribution?",
				Options: []PresetOption{
					{Key: "A", Label: "Homebrew"},
					{Key: "B", Label: "npm"},
					{Key: "C", Label: "pip / pipx"},
					{Key: "D", Label: "GitHub Releases (binaries)"},
					{Key: "E", Label: "Cargo"},
				},
				Impact:  6,
				Context: "How users install and update the tool",
			},
			{
				Category: "Config",
				Question: "Config file format?",
				Options: []PresetOption{
					{Key: "A", Label: "YAML"},
					{Key: "B", Label: "TOML"},
					{Key: "C", Label: "JSON"},
					{Key: "D", Label: "Env vars only"},
				},
				Impact:  4,
				Context: "User-facing configuration experience",
			},
			{
				Category: "Config",
				Question: "Config file location?",
				Options: []PresetOption{
					{Key: "A", Label: "XDG (~/.config/<app>)"},
					{Key: "B", Label: "Home directory (~/.<app>)"},
					{Key: "C", Label: "Project-local only"},
				},
				Impact:  3,
				Context: "Follows platform conventions",
			},
		},
	}
}

func webAppPreset() Preset {
	return Preset{
		Name:        "web-app",
		Description: "Common decisions for web application projects",
		Source:      "builtin",
		Decisions: []PresetDecision{
			{
				Category: "Stack",
				Question: "Frontend framework?",
				Options: []PresetOption{
					{Key: "A", Label: "React"},
					{Key: "B", Label: "Vue"},
					{Key: "C", Label: "Svelte"},
					{Key: "D", Label: "Angular"},
					{Key: "E", Label: "HTMX"},
				},
				Impact:  9,
				Context: "Foundational choice for all frontend code",
			},
			{
				Category: "Stack",
				Question: "Language / type system?",
				Options: []PresetOption{
					{Key: "A", Label: "TypeScript"},
					{Key: "B", Label: "JavaScript"},
				},
				Impact:  8,
				Context: "Affects tooling, IDE support, and refactoring safety",
			},
			{
				Category: "Styling",
				Question: "CSS approach?",
				Options: []PresetOption{
					{Key: "A", Label: "Tailwind CSS"},
					{Key: "B", Label: "CSS Modules"},
					{Key: "C", Label: "Styled Components"},
					{Key: "D", Label: "Plain CSS / Sass"},
				},
				Impact:  6,
				Context: "Determines how all UI styling is written",
			},
			{
				Category: "State",
				Question: "State management?",
				Options: []PresetOption{
					{Key: "A", Label: "Zustand"},
					{Key: "B", Label: "Redux Toolkit"},
					{Key: "C", Label: "Pinia"},
					{Key: "D", Label: "Jotai / Signals"},
					{Key: "E", Label: "Framework built-in"},
				},
				Impact:  7,
				Context: "How client-side state is organized",
			},
			{
				Category: "API",
				Question: "Backend / API layer?",
				Options: []PresetOption{
					{Key: "A", Label: "Next.js API routes"},
					{Key: "B", Label: "Separate REST API"},
					{Key: "C", Label: "tRPC"},
					{Key: "D", Label: "GraphQL"},
					{Key: "E", Label: "Firebase / Supabase"},
				},
				Impact:  8,
				Context: "Server-side architecture and data fetching strategy",
			},
			{
				Category: "Deploy",
				Question: "Hosting platform?",
				Options: []PresetOption{
					{Key: "A", Label: "Vercel"},
					{Key: "B", Label: "Netlify"},
					{Key: "C", Label: "Cloudflare Pages"},
					{Key: "D", Label: "AWS Amplify"},
					{Key: "E", Label: "Self-hosted"},
				},
				Impact:  6,
				Context: "Affects build pipeline and deployment workflow",
			},
			{
				Category: "Deploy",
				Question: "CI/CD approach?",
				Options: []PresetOption{
					{Key: "A", Label: "GitHub Actions"},
					{Key: "B", Label: "GitLab CI"},
					{Key: "C", Label: "Platform built-in (Vercel/Netlify)"},
				},
				Impact:  4,
				Context: "Automated testing and deployment pipeline",
			},
		},
	}
}
