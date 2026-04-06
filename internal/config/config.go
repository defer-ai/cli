package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all defer CLI configuration, loaded from global and project
// sources and optionally overridden by CLI flags.
type Config struct {
	Model       string                  `json:"model,omitempty"`
	Provider    string                  `json:"provider,omitempty"`
	APIKey      string                  `json:"apiKey,omitempty"`
	DefaultCare string                  `json:"defaultCare,omitempty"` // auto/review
	DomainCare  map[string]string       `json:"domainCare,omitempty"` // per-domain care level defaults
	Hooks       map[string][]HookConfig `json:"hooks,omitempty"`      // lifecycle hooks
	Skills      SkillsConfig            `json:"skills,omitempty"`     // skill directories
	MascotSize  string                  `json:"mascotSize,omitempty"` // "none", "small", "large"
	Theme       string                  `json:"theme,omitempty"`      // accent color name
}

// HookConfig describes a single lifecycle hook action.
type HookConfig struct {
	Command string `json:"command,omitempty"` // bash command to run
	URL     string `json:"url,omitempty"`     // webhook URL
}

// SkillsConfig holds additional skill directory paths.
type SkillsConfig struct {
	Dirs []string `json:"dirs,omitempty"` // additional skill directories
}

// GlobalConfigPath returns the path to the global config file (~/.defer/config.json).
func GlobalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".defer", "config.json")
	}
	return filepath.Join(home, ".defer", "config.json")
}

// ProjectConfigPath returns the path to the project-level config file.
func ProjectConfigPath(cwd string) string {
	return filepath.Join(cwd, ".defer", "config.json")
}

// LoadConfig loads and merges global + project configs. The project config
// values override global ones. Missing files are silently skipped.
func LoadConfig(cwd string) (*Config, error) {
	global, err := loadFile(GlobalConfigPath())
	if err != nil {
		return nil, err
	}

	project, err := loadFile(ProjectConfigPath(cwd))
	if err != nil {
		return nil, err
	}

	merged := merge(global, project)
	return merged, nil
}

// MergeWithFlags applies CLI flag overrides onto cfg. Only non-empty flag
// values are applied.
func MergeWithFlags(cfg *Config, model, provider, apiKey string) {
	if model != "" {
		cfg.Model = model
	}
	if provider != "" {
		cfg.Provider = provider
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
}

// SaveProjectConfig writes cfg to the project-level .defer/config.json,
// creating the directory if needed.
func SaveProjectConfig(cwd string, cfg *Config) error {
	p := ProjectConfigPath(cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

// LoadGlobalConfig loads only the global config (~/.defer/config.json).
// Returns nil if the file does not exist.
func LoadGlobalConfig() (*Config, error) {
	path := GlobalConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveGlobalConfig writes cfg to ~/.defer/config.json, creating the directory if needed.
func SaveGlobalConfig(cfg *Config) error {
	path := GlobalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// loadFile reads and parses a JSON config file. If the file does not exist,
// it returns a zero-value Config (not an error).
func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// merge combines base and override into a new Config.
// String fields: non-empty override wins.
// Maps: keys are merged; override wins per-key.
// Slices: concatenated (base first, then override).
func merge(base, override *Config) *Config {
	out := &Config{}

	// String fields — non-empty override wins.
	out.Model = mergeString(base.Model, override.Model)
	out.Provider = mergeString(base.Provider, override.Provider)
	out.APIKey = mergeString(base.APIKey, override.APIKey)
	out.DefaultCare = mergeString(base.DefaultCare, override.DefaultCare)
	out.MascotSize = mergeString(base.MascotSize, override.MascotSize)
	out.Theme = mergeString(base.Theme, override.Theme)

	// Map[string]string — merge keys.
	out.DomainCare = mergeMapString(base.DomainCare, override.DomainCare)

	// Map[string][]HookConfig — merge keys; per-key slices are concatenated.
	out.Hooks = mergeMapHooks(base.Hooks, override.Hooks)

	// SkillsConfig — concatenate dirs.
	out.Skills.Dirs = concatSlice(base.Skills.Dirs, override.Skills.Dirs)

	return out
}

func mergeString(base, override string) string {
	if override != "" {
		return override
	}
	return base
}

func mergeMapString(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func mergeMapHooks(base, override map[string][]HookConfig) map[string][]HookConfig {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string][]HookConfig)
	for k, v := range base {
		out[k] = append(out[k], v...)
	}
	for k, v := range override {
		out[k] = append(out[k], v...)
	}
	return out
}

func concatSlice(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}
