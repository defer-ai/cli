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

	b.WriteString("| ID | Category | Question | Answer | Source | Date |\n")
	b.WriteString("|----|----------|----------|--------|--------|------|\n")
	for _, d := range store.Decisions {
		answer := "(pending)"
		if d.Answer != nil {
			answer = *d.Answer
		}
		source := d.Source
		if source == "" {
			source = "user"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			d.ID, d.Category, d.Question, answer, source, d.Date))
	}

	b.WriteString("\n")
	return os.WriteFile(mdPath(cwd), []byte(b.String()), 0o644)
}
