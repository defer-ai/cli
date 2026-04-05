package decision

import "time"

// Feature represents a named feature that decisions can relate to.
type Feature struct {
	ID          string `json:"id"`                    // F-MSG, F-AUT
	Name        string `json:"name"`                  // "messaging", "auth"
	Description string `json:"description,omitempty"`
}

// DecisionOption is one possible answer for a decision.
type DecisionOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// ImportRef records where an imported decision came from.
type ImportRef struct {
	Project    string `json:"project"`    // source project path or URL
	OriginalID string `json:"originalID"` // ID in the source project
	ImportedAt string `json:"importedAt"` // RFC3339 timestamp
}

// Decision represents a single tracked decision.
type Decision struct {
	ID             string           `json:"id"`
	Category       string           `json:"category"`
	Question       string           `json:"question"`
	Options        []DecisionOption `json:"options"`
	Context        string           `json:"context"`
	Answer         *string          `json:"answer"`                        // nil = pending
	Delegated      bool             `json:"delegated"`
	Implicit       bool             `json:"implicit"`
	Reasoning      string           `json:"reasoning,omitempty"`
	Source         string           `json:"source,omitempty"`              // "user", "auto", "agent", "discovered"
	OriginalSource string           `json:"originalSource,omitempty"`      // set once when answer first assigned
	RevisionCount  int              `json:"revisionCount,omitempty"`       // incremented each time answer changes
	Impact         int              `json:"impact,omitempty"`              // 0-10, how many other decisions this affects
	DependsOn      []string         `json:"dependsOn,omitempty"`           // IDs of decisions this depends on
	Features       []string         `json:"features,omitempty"`            // feature tags (e.g. "auth", "onboarding")
	Date           string           `json:"date"`
	CreatedAt      string           `json:"createdAt,omitempty"`           // RFC3339 when decision was created
	AnsweredAt     string           `json:"answeredAt,omitempty"`          // RFC3339 when answer was last set
	ImportedFrom   *ImportRef       `json:"importedFrom,omitempty"`        // provenance for imported decisions
}

// DecisionStore is the top-level persisted structure.
type DecisionStore struct {
	Task      string     `json:"task"`
	Decisions []Decision `json:"decisions"`
	Features  []Feature  `json:"features,omitempty"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

// IsPending returns true if the decision has no answer yet.
func (d *Decision) IsPending() bool {
	return d.Answer == nil
}

// StrAnswer returns the answer string or empty.
func (d *Decision) StrAnswer() string {
	if d.Answer == nil {
		return ""
	}
	return *d.Answer
}

// SetAnswer updates the answer on a decision, tracking revision metadata.
// It sets OriginalSource on first answer, increments RevisionCount on changes,
// and updates AnsweredAt.
func (d *Decision) SetAnswer(answer, source string) {
	now := timeNow().UTC().Format(time.RFC3339)

	if d.Answer != nil {
		// Changing an existing answer
		d.RevisionCount++
	} else {
		// First time answering
		d.OriginalSource = source
	}

	d.Answer = &answer
	d.Source = source
	d.AnsweredAt = now
}

// MarkCreated sets CreatedAt if not already set.
func (d *Decision) MarkCreated() {
	if d.CreatedAt == "" {
		d.CreatedAt = timeNow().UTC().Format(time.RFC3339)
	}
}

// Ptr returns a pointer to a string (helper for setting answers).
func Ptr(s string) *string {
	return &s
}

// timeNow is a hook for testing.
var timeNow = time.Now
