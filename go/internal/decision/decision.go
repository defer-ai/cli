package decision

// DecisionOption is one possible answer for a decision.
type DecisionOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Decision represents a single tracked decision.
type Decision struct {
	ID        string           `json:"id"`
	Category  string           `json:"category"`
	Question  string           `json:"question"`
	Options   []DecisionOption `json:"options"`
	Context   string           `json:"context"`
	Answer    *string          `json:"answer"`              // nil = pending
	Delegated bool             `json:"delegated"`
	Implicit  bool             `json:"implicit"`
	Reasoning string           `json:"reasoning,omitempty"`
	Source    string           `json:"source,omitempty"` // "user", "auto", "agent", "discovered"
	Impact    int              `json:"impact,omitempty"`    // 0-10, how many other decisions this affects
	DependsOn []string         `json:"dependsOn,omitempty"` // IDs of decisions this depends on
	Date      string           `json:"date"`
}

// DecisionStore is the top-level persisted structure.
type DecisionStore struct {
	Task      string     `json:"task"`
	Decisions []Decision `json:"decisions"`
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

// Ptr returns a pointer to a string (helper for setting answers).
func Ptr(s string) *string {
	return &s
}
