package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/defer-ai/cli/internal/agent"
)

// Skill represents a loaded prompt/process file.
type Skill struct {
	Name        string
	Description string
	Prompt      string            // the actual prompt text
	Metadata    map[string]string // parsed YAML frontmatter
	Path        string            // where it was loaded from
}

// LoadSkills discovers and loads skills from .defer/skills/ directories.
// Walks up from cwd to find skill dirs, deeper paths override shallower.
func LoadSkills(cwd string) ([]Skill, error) {
	// Collect all .defer/skills/ directories from cwd up to root.
	// Deeper (closer to cwd) paths override shallower ones.
	var dirs []string
	dir := filepath.Clean(cwd)
	for {
		candidate := filepath.Join(dir, ".defer", "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			dirs = append(dirs, candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Reverse so shallower dirs are processed first (deeper override later)
	for i, j := 0, len(dirs)-1; i < j; i, j = i+1, j-1 {
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}

	skillMap := make(map[string]Skill)
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(d, entry.Name())
			skill, err := LoadSkillFile(path)
			if err != nil {
				continue
			}
			skillMap[skill.Name] = *skill
		}
	}

	// Convert to sorted slice
	var result []Skill
	for _, s := range skillMap {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// LoadSkillFile loads a single skill from a .md file with YAML frontmatter.
// Frontmatter is delimited by --- lines. Key: value pairs are parsed from it.
func LoadSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	metadata := make(map[string]string)
	prompt := content

	// Parse frontmatter: content between first and second "---" lines
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Find the opening delimiter
		trimmed := strings.TrimSpace(content)
		afterFirst := strings.TrimPrefix(trimmed, "---")
		afterFirst = strings.TrimLeft(afterFirst, " \t")
		if len(afterFirst) > 0 && afterFirst[0] == '\n' {
			afterFirst = afterFirst[1:]
		} else if len(afterFirst) > 1 && afterFirst[0] == '\r' && afterFirst[1] == '\n' {
			afterFirst = afterFirst[2:]
		}

		// Find the closing delimiter
		closeIdx := strings.Index(afterFirst, "\n---")
		if closeIdx >= 0 {
			frontmatter := afterFirst[:closeIdx]
			rest := afterFirst[closeIdx+4:] // skip \n---

			// Skip the rest of the --- line
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				rest = rest[nl+1:]
			} else {
				rest = ""
			}

			// Parse key: value lines
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				colonIdx := strings.Index(line, ":")
				if colonIdx > 0 {
					key := strings.TrimSpace(line[:colonIdx])
					value := strings.TrimSpace(line[colonIdx+1:])
					metadata[key] = value
				}
			}

			prompt = strings.TrimSpace(rest)
		}
	}

	// Derive name from metadata or filename
	name := metadata["name"]
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &Skill{
		Name:        name,
		Description: metadata["description"],
		Prompt:      prompt,
		Metadata:    metadata,
		Path:        path,
	}, nil
}

// DefaultSkills returns the built-in skills (decompose, plan, execute, extract, verify, scan).
func DefaultSkills() map[string]Skill {
	return map[string]Skill{
		"decompose": {
			Name:        "decompose",
			Description: "Break task into decisions",
			Prompt:      agent.DecomposePrompt,
			Metadata: map[string]string{
				"name":        "decompose",
				"description": "Break task into decisions",
				"when-to-use": "When starting a new task",
			},
		},
		"plan": {
			Name:        "plan",
			Description: "Identify remaining implementation decisions",
			Prompt:      agent.PlanPrompt,
			Metadata: map[string]string{
				"name":        "plan",
				"description": "Identify remaining implementation decisions",
				"when-to-use": "After decompose, to fill in missing decisions",
			},
		},
		"execute": {
			Name:        "execute",
			Description: "Implement a domain given decisions",
			Prompt:      agent.ExecutePromptTemplate,
			Metadata: map[string]string{
				"name":        "execute",
				"description": "Implement a domain given decisions",
				"when-to-use": "When all decisions for a domain are answered",
			},
		},
		"extract": {
			Name:        "extract",
			Description: "Extract decisions from implementation",
			Prompt:      agent.ExtractPrompt,
			Metadata: map[string]string{
				"name":        "extract",
				"description": "Extract decisions from implementation",
				"when-to-use": "After execution to catalog implicit decisions",
			},
		},
		"verify": {
			Name:        "verify",
			Description: "Review domain implementation for correctness",
			Prompt:      agent.VerifyPrompt,
			Metadata: map[string]string{
				"name":        "verify",
				"description": "Review domain implementation for correctness",
				"when-to-use": "After execution to check for errors",
			},
		},
		"scan": {
			Name:        "scan",
			Description: "Analyze existing codebase to discover decisions",
			Prompt:      agent.ScanPrompt,
			Metadata: map[string]string{
				"name":        "scan",
				"description": "Analyze existing codebase to discover decisions",
				"when-to-use": "When scanning an existing project",
			},
		},
	}
}
