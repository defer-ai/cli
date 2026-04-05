package decision

import (
	"strings"
	"testing"
)

func TestCategoryPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Single word: first 3 letters
		{"Stack", "STA"},
		{"Data", "DAT"},
		{"Security", "SEC"},
		{"Auth", "AUT"},
		{"Architecture", "ARC"},
		{"Deployment", "DEP"},
		{"Protocol", "PRO"},
		{"Rooms", "ROO"},
		{"Persistence", "PER"},
		{"Features", "FEA"},
		{"Misc", "MIS"},
		{"API", "API"},

		// Short words: pad by repeating last char
		{"UI", "UII"},
		{"A", "AAA"},
		{"DB", "DBB"},

		// Multi-word: first letter of each word
		{"UI Polish", "UPO"},
		{"End to End", "ETE"},
		{"User Interface", "UIN"},
		{"Data Model Design", "DMD"},
		{"Build and Deploy", "BAD"},

		// Multi-word with only 2 words (2 initials, pad from last)
		{"Build Deploy", "BDE"},

		// Special chars stripped
		{"Build & Deploy", "BDE"},

		// Empty / blank
		{"", "UNK"},
		{"   ", "UNK"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := categoryPrefix(tt.input)
			if got != tt.want {
				t.Errorf("categoryPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNextIDBasic(t *testing.T) {
	// No existing decisions
	id := NextID(nil, "Stack")
	if id != "STA-0001" {
		t.Errorf("NextID(nil, Stack) = %q, want STA-0001", id)
	}
}

func TestNextIDIncrementsFromExisting(t *testing.T) {
	existing := []Decision{
		{ID: "STA-0001"},
		{ID: "STA-0002"},
		{ID: "DAT-0001"},
	}
	id := NextID(existing, "Stack")
	if id != "STA-0003" {
		t.Errorf("NextID with existing STA-0001,0002 = %q, want STA-0003", id)
	}

	id = NextID(existing, "Data")
	if id != "DAT-0002" {
		t.Errorf("NextID with existing DAT-0001 = %q, want DAT-0002", id)
	}

	id = NextID(existing, "Security")
	if id != "SEC-0001" {
		t.Errorf("NextID for new category = %q, want SEC-0001", id)
	}
}

func TestNextIDDifferentCategories(t *testing.T) {
	id1 := NextID(nil, "Stack")
	id2 := NextID(nil, "Data")

	if id1 == id2 {
		t.Errorf("IDs from different categories should differ: %s", id1)
	}

	if !strings.HasPrefix(id1, "STA-") {
		t.Errorf("id1 prefix wrong: %s", id1)
	}
	if !strings.HasPrefix(id2, "DAT-") {
		t.Errorf("id2 prefix wrong: %s", id2)
	}
}

func TestNextIDFormat(t *testing.T) {
	id := NextID(nil, "Stack")

	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		t.Fatalf("ID %q has %d parts, want PREFIX-SEQ", id, len(parts))
	}

	prefix := parts[0]
	if len(prefix) != 3 {
		t.Errorf("prefix %q length = %d, want 3", prefix, len(prefix))
	}
	if prefix != strings.ToUpper(prefix) {
		t.Errorf("prefix %q should be uppercase", prefix)
	}

	seq := parts[1]
	if len(seq) != 4 {
		t.Errorf("seq %q has %d chars, want 4", seq, len(seq))
	}
}

func TestNextIDZeroPadding(t *testing.T) {
	tests := []struct {
		existing []Decision
		want     string
	}{
		{nil, "STA-0001"},
		{[]Decision{{ID: "STA-0001"}}, "STA-0002"},
		{[]Decision{{ID: "STA-0009"}}, "STA-0010"},
		{[]Decision{{ID: "STA-0099"}}, "STA-0100"},
		{[]Decision{{ID: "STA-0999"}}, "STA-1000"},
		{[]Decision{{ID: "STA-9999"}}, "STA-10000"},
	}
	for _, tt := range tests {
		got := NextID(tt.existing, "Stack")
		if got != tt.want {
			t.Errorf("NextID = %q, want %q", got, tt.want)
		}
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Stack", 1},
		{"UI Polish", 2},
		{"Build & Deploy", 2},
		{"Build-Deploy", 2},
		{"Build_Deploy", 2},
		{"", 0},
		{"   ", 0},
		{"one-two three_four", 4},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitWords(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitWords(%q) = %v (len %d), want len %d", tt.input, got, len(got), tt.want)
			}
		})
	}
}

func TestFeatureID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"messaging", "MES"},
		{"auth", "AUT"},
		{"encryption", "ENC"},
		{"UI", "UII"},
		{"user interface", "UIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FeatureID(tt.name)
			if got != tt.want {
				t.Errorf("FeatureID(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsPending(t *testing.T) {
	d := Decision{Answer: nil}
	if !d.IsPending() {
		t.Error("expected pending")
	}

	answer := "TypeScript"
	d.Answer = &answer
	if d.IsPending() {
		t.Error("expected not pending")
	}
}
