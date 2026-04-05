package keybindings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Action identifies a bindable action.
type Action string

const (
	ActionNavigateUp   Action = "navigate.up"
	ActionNavigateDown Action = "navigate.down"
	ActionInspect      Action = "inspect"
	ActionBack         Action = "back"
	ActionSearch       Action = "search"
	ActionChat         Action = "chat"
	ActionCustom       Action = "custom"
	ActionShuffle      Action = "shuffle"
	ActionWhy          Action = "why"
	ActionAsk          Action = "ask"
	ActionConfirm      Action = "confirm"
	ActionCancel       Action = "cancel"
	ActionQuit         Action = "quit"
	ActionCareUp       Action = "care.up"
	ActionCareDown     Action = "care.down"
)

// Bindings maps actions to key strings.
type Bindings map[Action][]string

// DefaultBindings returns vim-style defaults.
func DefaultBindings() Bindings {
	return Bindings{
		ActionNavigateUp:   {"up"},
		ActionNavigateDown: {"down"},
		ActionInspect:      {"enter"},
		ActionBack:         {"q", "esc"},
		ActionSearch:       {"/"},
		ActionChat:         {"tab"},
		ActionCustom:       {"c"},
		ActionShuffle:      {"s"},
		ActionWhy:          {"w"},
		ActionAsk:          {"a"},
		ActionConfirm:      {"enter"},
		ActionCancel:       {"esc"},
		ActionQuit:         {"ctrl+c"},
		ActionCareUp:       {"l", "right"},
		ActionCareDown:     {"h", "left"},
	}
}

// configPath returns the keybindings config file path.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".defer", "keybindings.json")
	}
	return filepath.Join(home, ".defer", "keybindings.json")
}

// LoadBindings loads from ~/.defer/keybindings.json, merging with defaults.
// User bindings replace defaults per-action (not append).
// If the file does not exist or is unreadable, defaults are returned.
func LoadBindings() Bindings {
	return LoadBindingsFrom(configPath())
}

// LoadBindingsFrom loads keybindings from a specific file path, merging with
// defaults. Exported for testing.
func LoadBindingsFrom(path string) Bindings {
	b := DefaultBindings()

	data, err := os.ReadFile(path)
	if err != nil {
		return b
	}

	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return b
	}

	for actionStr, keys := range raw {
		action := Action(actionStr)
		// Only override if we know this action (present in defaults) or it is
		// a valid-looking action string. We accept any action string so users
		// can define custom actions too.
		b[action] = keys
	}

	return b
}

// Resolve returns the Action for a key string, or "" if not bound.
// When multiple actions bind the same key, the first match in iteration
// order is returned (map order is non-deterministic, but in practice
// collisions are rare and intentional -- e.g. "enter" is both inspect
// and confirm depending on context).
func (b Bindings) Resolve(key string) Action {
	for action, keys := range b {
		for _, k := range keys {
			if k == key {
				return action
			}
		}
	}
	return ""
}

// ResolveAll returns all actions bound to a key string.
func (b Bindings) ResolveAll(key string) []Action {
	var actions []Action
	for action, keys := range b {
		for _, k := range keys {
			if k == key {
				actions = append(actions, action)
				break
			}
		}
	}
	return actions
}
