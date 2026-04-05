package decision

import (
	"fmt"
	"strings"
)

// DiffEntry represents a single change between two decision stores.
type DiffEntry struct {
	Type     string // "added", "removed", "changed", "invalidated"
	ID       string
	Category string
	Question string
	OldValue string // for "changed" type
	NewValue string // for "changed" type
}

// DiffStores compares two decision stores and returns the differences.
// If old is nil, all decisions in new are treated as additions.
func DiffStores(old, new *DecisionStore) []DiffEntry {
	var entries []DiffEntry

	oldByID := map[string]*Decision{}
	if old != nil {
		for i := range old.Decisions {
			oldByID[old.Decisions[i].ID] = &old.Decisions[i]
		}
	}

	newByID := map[string]*Decision{}
	for i := range new.Decisions {
		newByID[new.Decisions[i].ID] = &new.Decisions[i]
	}

	// Check for additions and changes
	for _, d := range new.Decisions {
		oldD, existed := oldByID[d.ID]
		if !existed {
			entries = append(entries, DiffEntry{
				Type:     "added",
				ID:       d.ID,
				Category: d.Category,
				Question: d.Question,
				NewValue: d.StrAnswer(),
			})
			continue
		}

		// Check for answer changes
		oldAnswer := oldD.StrAnswer()
		newAnswer := d.StrAnswer()
		if oldAnswer != newAnswer {
			diffType := "changed"
			if d.Source == "invalidated" {
				diffType = "invalidated"
			}
			entries = append(entries, DiffEntry{
				Type:     diffType,
				ID:       d.ID,
				Category: d.Category,
				Question: d.Question,
				OldValue: oldAnswer,
				NewValue: newAnswer,
			})
		}
	}

	// Check for removals
	if old != nil {
		for _, d := range old.Decisions {
			if _, exists := newByID[d.ID]; !exists {
				entries = append(entries, DiffEntry{
					Type:     "removed",
					ID:       d.ID,
					Category: d.Category,
					Question: d.Question,
					OldValue: d.StrAnswer(),
				})
			}
		}
	}

	return entries
}

// FormatDiff renders diff entries as a markdown-formatted string suitable for
// GitHub PR comments.
func FormatDiff(entries []DiffEntry, task string) string {
	if len(entries) == 0 {
		return "No decision changes."
	}

	var b strings.Builder
	b.WriteString("## Architectural Decisions Changed\n\n")
	if task != "" {
		b.WriteString(fmt.Sprintf("> Task: %s\n\n", task))
	}

	b.WriteString("| | ID | Category | Question | Change |\n")
	b.WriteString("|---|---|----------|----------|--------|\n")

	for _, e := range entries {
		icon := ""
		change := ""
		switch e.Type {
		case "added":
			icon = "+"
			if e.NewValue != "" {
				change = fmt.Sprintf("Added → %s", e.NewValue)
			} else {
				change = "Added (pending)"
			}
		case "removed":
			icon = "-"
			change = fmt.Sprintf("Removed (was: %s)", e.OldValue)
		case "changed":
			icon = "~"
			if e.OldValue == "" {
				change = fmt.Sprintf("Decided → %s", e.NewValue)
			} else {
				change = fmt.Sprintf("%s → %s", e.OldValue, e.NewValue)
			}
		case "invalidated":
			icon = "!"
			change = fmt.Sprintf("Invalidated (was: %s)", e.OldValue)
		}
		b.WriteString(fmt.Sprintf("| %s | @%s | %s | %s | %s |\n",
			icon, e.ID, e.Category, e.Question, change))
	}

	return b.String()
}
