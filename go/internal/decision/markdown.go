package decision

import (
	"fmt"
	"os"
	"strings"
)

// GenerateMarkdown writes DECISIONS.md from the store.
func GenerateMarkdown(cwd string, store *DecisionStore) error {
	var b strings.Builder

	b.WriteString("# DECISIONS.md\n\n")
	b.WriteString(fmt.Sprintf("> Task: %s\n\n", store.Task))

	// User decisions
	var userDecs, aiDecs []Decision
	for _, d := range store.Decisions {
		if d.Implicit {
			aiDecs = append(aiDecs, d)
		} else {
			userDecs = append(userDecs, d)
		}
	}

	b.WriteString("## Decisions\n\n")
	b.WriteString("| ID | Category | Question | Answer | Date |\n")
	b.WriteString("|----|----------|----------|--------|------|\n")
	for _, d := range userDecs {
		answer := "(pending)"
		if d.Answer != nil {
			if d.Delegated {
				answer = "DELEGATED: " + *d.Answer
			} else {
				answer = *d.Answer
			}
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", d.ID, d.Category, d.Question, answer, d.Date))
	}

	if len(aiDecs) > 0 {
		b.WriteString("\n## AI Choices\n\n")
		b.WriteString("| ID | Category | What was decided | Reasoning |\n")
		b.WriteString("|----|----------|------------------|----------|\n")
		for _, d := range aiDecs {
			answer := d.StrAnswer()
			if answer == "" {
				answer = d.Question
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", d.ID, d.Category, answer, d.Reasoning))
		}
	}

	b.WriteString("\n")
	return os.WriteFile(mdPath(cwd), []byte(b.String()), 0o644)
}
