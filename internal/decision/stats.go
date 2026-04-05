package decision

import (
	"fmt"
	"sort"
	"strings"
)

// Stats holds computed analytics for a decision store.
type Stats struct {
	Total         int
	Answered      int
	Pending       int
	Delegated     int
	AutoCount     int
	UserCount     int
	OverrideCount int // decisions where OriginalSource="auto" but Source="user"
	OverrideTotal int // total auto-decided (denominator for override rate)
	ByCategory    map[string]CategoryStats
	ImpactHigh    int // impact 7-10
	ImpactMedium  int // impact 4-6
	ImpactLow     int // impact 0-3
	MaxDepthChain string
	MaxDepth      int
	MostRevised   []RevisedDecision
}

// CategoryStats holds per-category counts.
type CategoryStats struct {
	Total    int
	Answered int
	Pending  int
}

// RevisedDecision records a decision that has been revised.
type RevisedDecision struct {
	ID            string
	Question      string
	RevisionCount int
}

// ComputeStats computes all analytics metrics from a decision store.
func ComputeStats(store *DecisionStore) Stats {
	s := Stats{
		ByCategory: make(map[string]CategoryStats),
	}

	if store == nil {
		return s
	}

	s.Total = len(store.Decisions)

	for _, d := range store.Decisions {
		// Answered vs pending
		if d.Answer != nil {
			s.Answered++
		} else {
			s.Pending++
		}

		// Delegated
		if d.Delegated {
			s.Delegated++
		}

		// Source counts (only for answered decisions)
		if d.Answer != nil {
			switch d.Source {
			case "auto":
				s.AutoCount++
			case "user":
				s.UserCount++
			}
		}

		// Override detection: originally auto, now user
		if d.OriginalSource == "auto" {
			s.OverrideTotal++
			if d.Source == "user" {
				s.OverrideCount++
			}
		}

		// Category grouping
		cat := d.Category
		if cat == "" {
			cat = "(none)"
		}
		cs := s.ByCategory[cat]
		cs.Total++
		if d.Answer != nil {
			cs.Answered++
		} else {
			cs.Pending++
		}
		s.ByCategory[cat] = cs

		// Impact distribution
		switch {
		case d.Impact >= 7:
			s.ImpactHigh++
		case d.Impact >= 4:
			s.ImpactMedium++
		default:
			s.ImpactLow++
		}
	}

	// Dependency depth
	s.MaxDepth, s.MaxDepthChain = computeMaxDepth(store.Decisions)

	// Most revised (top 3)
	s.MostRevised = computeMostRevised(store.Decisions, 3)

	return s
}

// computeMaxDepth finds the longest dependency chain using BFS from each root.
// Depth is measured as the number of edges in the longest chain.
func computeMaxDepth(decisions []Decision) (int, string) {
	// Build adjacency: parent -> children (via DependsOn)
	// If B.DependsOn contains A, then A -> B is an edge.
	children := make(map[string][]string)
	idSet := make(map[string]bool)
	hasParent := make(map[string]bool)

	for _, d := range decisions {
		idSet[d.ID] = true
		for _, dep := range d.DependsOn {
			children[dep] = append(children[dep], d.ID)
			hasParent[d.ID] = true
		}
	}

	// Find roots (nodes with no parents)
	var roots []string
	for _, d := range decisions {
		if !hasParent[d.ID] {
			roots = append(roots, d.ID)
		}
	}

	bestDepth := 0
	var bestChain []string

	// BFS from each root tracking depth and path
	for _, root := range roots {
		type entry struct {
			id    string
			depth int
			chain []string
		}
		queue := []entry{{id: root, depth: 0, chain: []string{root}}}
		visited := map[string]bool{root: true}

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			if cur.depth > bestDepth {
				bestDepth = cur.depth
				bestChain = cur.chain
			}

			for _, child := range children[cur.id] {
				if !visited[child] {
					visited[child] = true
					newChain := make([]string, len(cur.chain)+1)
					copy(newChain, cur.chain)
					newChain[len(cur.chain)] = child
					queue = append(queue, entry{
						id:    child,
						depth: cur.depth + 1,
						chain: newChain,
					})
				}
			}
		}
	}

	chainStr := ""
	if len(bestChain) > 0 {
		chainStr = strings.Join(bestChain, " → ")
	}

	return bestDepth, chainStr
}

// computeMostRevised returns the top N decisions by RevisionCount, skipping those with 0.
func computeMostRevised(decisions []Decision, n int) []RevisedDecision {
	var revised []RevisedDecision
	for _, d := range decisions {
		if d.RevisionCount > 0 {
			revised = append(revised, RevisedDecision{
				ID:            d.ID,
				Question:      d.Question,
				RevisionCount: d.RevisionCount,
			})
		}
	}

	sort.Slice(revised, func(i, j int) bool {
		return revised[i].RevisionCount > revised[j].RevisionCount
	})

	if len(revised) > n {
		revised = revised[:n]
	}

	return revised
}

// FormatStats returns a human-readable string representation of Stats.
func FormatStats(s Stats) string {
	var b strings.Builder

	// Summary line
	b.WriteString(fmt.Sprintf("Decisions: %d total (%d decided, %d pending)\n",
		s.Total, s.Answered, s.Pending))

	// Auto/User breakdown (only if there are answered decisions with auto or user source)
	autoUserTotal := s.AutoCount + s.UserCount
	if autoUserTotal > 0 {
		autoPct := 100 * s.AutoCount / autoUserTotal
		userPct := 100 * s.UserCount / autoUserTotal
		b.WriteString(fmt.Sprintf("Auto/Review: %d auto (%d%%), %d user (%d%%)\n",
			s.AutoCount, autoPct, s.UserCount, userPct))
	}

	// Override rate
	if s.OverrideTotal > 0 {
		overridePct := 100 * s.OverrideCount / s.OverrideTotal
		b.WriteString(fmt.Sprintf("Override rate: %d/%d auto-decisions overridden (%d%%)\n",
			s.OverrideCount, s.OverrideTotal, overridePct))
	}

	// By category
	if len(s.ByCategory) > 0 {
		b.WriteString("\nBy category:\n")
		// Sort categories for stable output
		cats := make([]string, 0, len(s.ByCategory))
		for cat := range s.ByCategory {
			cats = append(cats, cat)
		}
		sort.Strings(cats)

		for _, cat := range cats {
			cs := s.ByCategory[cat]
			detail := ""
			if cs.Pending == 0 {
				detail = "(all decided)"
			} else {
				detail = fmt.Sprintf("(%d decided, %d pending)", cs.Answered, cs.Pending)
			}
			b.WriteString(fmt.Sprintf("  %-10s %d decisions  %s\n", cat, cs.Total, detail))
		}
	}

	// Impact distribution
	b.WriteString("\nImpact distribution:\n")
	b.WriteString(fmt.Sprintf("  High (7-10):   %d decisions\n", s.ImpactHigh))
	b.WriteString(fmt.Sprintf("  Medium (4-6):  %d decisions\n", s.ImpactMedium))
	b.WriteString(fmt.Sprintf("  Low (0-3):     %d decisions\n", s.ImpactLow))

	// Dependency depth
	b.WriteString(fmt.Sprintf("\nDependency depth: max %d\n", s.MaxDepth))

	// Most revised
	if len(s.MostRevised) > 0 {
		parts := make([]string, len(s.MostRevised))
		for i, r := range s.MostRevised {
			parts[i] = fmt.Sprintf("@%s (%d revisions)", r.ID, r.RevisionCount)
		}
		b.WriteString(fmt.Sprintf("Most-revised: %s\n", strings.Join(parts, ", ")))
	}

	return b.String()
}
