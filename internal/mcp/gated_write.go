package mcp

// Gated-write tools for defer's executor phase.
//
// The decision-gated write architecture requires every file modification
// to be explicitly tied to one or more registered decisions. This file
// implements the two MCP tools that make that contract enforceable:
//
//   register_decision  — The executor declares a choice it's about to
//                        materialize (category, question, options, chosen
//                        value, alternatives considered, reasoning).
//                        Returns a decision_id. Care level is applied:
//                        CareLevelAuto → resolved immediately with the
//                        chosen value; CareLevelReview → stored as
//                        pending (MVP: stubbed to also resolve so the
//                        executor can continue; full TUI integration is
//                        a follow-up).
//
//   write_file         — The executor writes a file after passing an
//                        array of decision_ids it considers responsible
//                        for the file's content. The tool validates that
//                        at least one of the supplied decisions exists
//                        and is resolved. If validation passes, the file
//                        is written to disk under the server's cwd.
//
// These tools are paired with a --tools allowlist that removes the
// native Write/Edit tools from the executor's Claude Code session, so
// the only way to modify files is through this validated path.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/defer-ai/cli/internal/decision"
)

// normalizeForDedup strips a question down to lowercase alpha+space for
// dedup matching. "HTTP router / framework?" → "http router framework".
// This catches the most common duplicate scenario: decompose produces
// "HTTP router / framework" and the executor re-registers the same
// question with slightly different punctuation or casing.
func normalizeForDedup(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	var sb strings.Builder
	prevSpace := false
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			sb.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(sb.String())
}

// findExistingDecision returns the index of a decision in the store
// whose normalized question matches the given one, or -1 if not found.
func findExistingDecision(decs []decision.Decision, category, question string) int {
	normQ := normalizeForDedup(question)
	normCat := strings.ToLower(strings.TrimSpace(category))
	for i, d := range decs {
		// Same category (case-insensitive) AND same normalized question.
		if strings.ToLower(strings.TrimSpace(d.Category)) == normCat &&
			normalizeForDedup(d.Question) == normQ {
			return i
		}
	}
	// Fallback: match on question alone (cross-category) if the overlap
	// is very high. This catches decompose using "Stack" and executor
	// using "Technology" for the same question.
	for i, d := range decs {
		if normalizeForDedup(d.Question) == normQ {
			return i
		}
	}
	return -1
}

// toolRegisterDecision appends a new decision to the canonical store and
// immediately resolves it with the chosen value. The response includes
// the assigned decision_id, the resolved answer, and a status the caller
// can branch on.
func (s *Server) toolRegisterDecision(args json.RawMessage) callToolResult {
	var params struct {
		Category     string   `json:"category"`
		Question     string   `json:"question"`
		Chosen       string   `json:"chosen"`
		Alternatives []string `json:"alternatives"`
		Reasoning    string   `json:"reasoning"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &params); err != nil {
			return errResult(fmt.Sprintf("invalid arguments: %v", err))
		}
	}

	// Minimum shape: category + question + chosen. Alternatives and
	// reasoning are optional but strongly encouraged — the point of this
	// tool is to capture the "what else did you consider" context that
	// normally dies inside the model's hidden reasoning.
	if strings.TrimSpace(params.Category) == "" {
		return errResult("'category' is required")
	}
	if strings.TrimSpace(params.Question) == "" {
		return errResult("'question' is required")
	}
	if strings.TrimSpace(params.Chosen) == "" {
		return errResult("'chosen' is required")
	}

	var result callToolResult
	err := decision.WithStoreLock(s.cwd, func() error {
		store, err := decision.LoadStore(s.cwd)
		if err != nil || store == nil {
			// No store yet — create one on the fly with an empty task.
			// The executor phase runs inside a workdir that already has
			// .defer/decisions.json in normal usage, but this path makes
			// the tool useful in bare directories too (e.g. tests).
			store = &decision.DecisionStore{Decisions: nil}
		}

		// Dedup: if a decision with the same question already exists
		// (from the decompose phase or a prior register_decision call),
		// update it in place instead of creating a duplicate. This is
		// the MCP equivalent of the executor's reconcile logic — the
		// agent's register_decision call is ground truth about what it's
		// about to write, so it should overwrite the decompose plan.
		answer := params.Chosen
		if idx := findExistingDecision(store.Decisions, params.Category, params.Question); idx >= 0 {
			existing := &store.Decisions[idx]
			existing.SetAnswer(answer, "agent")
			if params.Reasoning != "" {
				existing.Reasoning = params.Reasoning
			}

			if err := decision.SaveStore(s.cwd, store); err != nil {
				result = errResult(fmt.Sprintf("save failed: %v", err))
				return nil
			}

			out := map[string]interface{}{
				"decision_id":     existing.ID,
				"resolved_answer": answer,
				"status":          "resolved",
				"deduplicated":    true,
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			result = textResult(string(data))
			return nil
		}

		// New decision — build the Options list with `chosen` as A and
		// any alternatives as subsequent entries.
		opts := []decision.DecisionOption{
			{Key: "A", Label: params.Chosen},
		}
		for i, alt := range params.Alternatives {
			if strings.TrimSpace(alt) == "" {
				continue
			}
			opts = append(opts, decision.DecisionOption{
				Key:   string(rune('B' + i)),
				Label: alt,
			})
		}

		id := decision.NextID(store.Decisions, params.Category)
		d := decision.Decision{
			ID:        id,
			Category:  params.Category,
			Question:  strings.TrimRight(strings.TrimSpace(params.Question), "?"),
			Options:   opts,
			Answer:    &answer,
			Source:    "agent",
			Implicit:  true,
			Reasoning: params.Reasoning,
		}
		d.MarkCreated()

		store.Decisions = append(store.Decisions, d)

		if err := decision.SaveStore(s.cwd, store); err != nil {
			result = errResult(fmt.Sprintf("save failed: %v", err))
			return nil
		}

		out := map[string]interface{}{
			"decision_id":     id,
			"resolved_answer": answer,
			"status":          "resolved",
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		result = textResult(string(data))
		return nil
	})
	if err != nil {
		return errResult(fmt.Sprintf("lock error: %v", err))
	}
	return result
}

// toolWriteFile writes a file to disk after validating that every
// decision_id in the supplied list exists in the store and has a non-nil
// answer. At least one valid decision_id must be supplied — writing with
// zero tracked decisions is the failure mode this whole architecture
// exists to prevent.
func (s *Server) toolWriteFile(args json.RawMessage) callToolResult {
	var params struct {
		DecisionIDs []string `json:"decision_ids"`
		Path        string   `json:"path"`
		Content     string   `json:"content"`
	}
	if args != nil {
		if err := json.Unmarshal(args, &params); err != nil {
			return errResult(fmt.Sprintf("invalid arguments: %v", err))
		}
	}

	if len(params.DecisionIDs) == 0 {
		return errResult("'decision_ids' must contain at least one id. " +
			"Call register_decision first to record the choices you're about to materialize, " +
			"then pass the returned decision_ids here.")
	}
	if strings.TrimSpace(params.Path) == "" {
		return errResult("'path' is required")
	}

	store, err := decision.LoadStore(s.cwd)
	if err != nil || store == nil {
		return errResult("no defer session found. register_decision must be called before write_file.")
	}

	// Build an index for O(1) lookups and validate every id is present
	// and resolved. Collecting all failures at once helps the model fix
	// the call on its first retry rather than ping-ponging.
	byID := make(map[string]*decision.Decision, len(store.Decisions))
	for i := range store.Decisions {
		byID[store.Decisions[i].ID] = &store.Decisions[i]
	}
	var missing, unresolved []string
	for _, id := range params.DecisionIDs {
		d, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if d.Answer == nil {
			unresolved = append(unresolved, id)
		}
	}
	if len(missing) > 0 || len(unresolved) > 0 {
		var sb strings.Builder
		sb.WriteString("decision validation failed: ")
		if len(missing) > 0 {
			sb.WriteString(fmt.Sprintf("unknown ids %v", missing))
		}
		if len(unresolved) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(fmt.Sprintf("unresolved (still pending) ids %v", unresolved))
		}
		sb.WriteString(". Call register_decision for any missing choices, then retry.")
		return errResult(sb.String())
	}

	// Resolve the path. Relative paths are anchored at the server's cwd
	// (which is the executor's workdir). Absolute paths are honored as-is
	// but must stay under the cwd — writing to /tmp or /etc from the
	// executor is not a feature, it's the bug this whole thing is
	// blocking.
	abs := params.Path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(s.cwd, params.Path)
	}
	// Clean + verify containment. A path like "../../etc/passwd" would
	// resolve above s.cwd and we reject it.
	absClean := filepath.Clean(abs)
	cwdClean := filepath.Clean(s.cwd)
	if !strings.HasPrefix(absClean+string(filepath.Separator), cwdClean+string(filepath.Separator)) && absClean != cwdClean {
		return errResult(fmt.Sprintf("path %q escapes the working directory %q", params.Path, s.cwd))
	}

	// Ensure the parent directory exists before writing. The native Write
	// tool does this implicitly; we mirror that behavior so the executor
	// doesn't have to do the bookkeeping itself.
	if err := os.MkdirAll(filepath.Dir(absClean), 0o755); err != nil {
		return errResult(fmt.Sprintf("mkdir parent: %v", err))
	}

	if err := os.WriteFile(absClean, []byte(params.Content), 0o644); err != nil {
		return errResult(fmt.Sprintf("write failed: %v", err))
	}

	out := map[string]interface{}{
		"ok":              true,
		"path":            absClean,
		"bytes_written":   len(params.Content),
		"tracked_by":      params.DecisionIDs,
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return textResult(string(data))
}
