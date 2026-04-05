package decision

import (
	"fmt"
	"strings"
)

// Trailers returns git trailer lines for all decided decisions.
// Format: "Decision-Ref: @STA-0001" (one per decided decision)
func Trailers(store *DecisionStore) string {
	if store == nil {
		return ""
	}
	var lines []string
	for _, d := range store.Decisions {
		if d.Answer != nil {
			lines = append(lines, fmt.Sprintf("Decision-Ref: @%s", d.ID))
		}
	}
	return strings.Join(lines, "\n")
}

// TrailersForIDs returns trailers for specific decision IDs only.
// Only decided decisions matching the given IDs are included.
func TrailersForIDs(store *DecisionStore, ids []string) string {
	if store == nil || len(ids) == 0 {
		return ""
	}
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var lines []string
	for _, d := range store.Decisions {
		if d.Answer != nil && idSet[d.ID] {
			lines = append(lines, fmt.Sprintf("Decision-Ref: @%s", d.ID))
		}
	}
	return strings.Join(lines, "\n")
}
