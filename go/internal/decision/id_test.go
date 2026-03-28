package decision

import (
	"strings"
	"sync"
	"testing"
)

func TestNextIDUnique(t *testing.T) {
	// Generate multiple IDs -- all must be unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NextID(nil, "Stack")
		if ids[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestNextIDPrefix(t *testing.T) {
	tests := []struct {
		category   string
		wantPrefix string
	}{
		{"Stack", "STACK-"},
		{"Data", "DATA-"},
		{"User Interface", "UI-"},
		{"Authentication", "AUTH-"},
		{"UI", "UI-"},
		{"Misc", "MISC-"},
		{"API", "API-"},
		{"A", "A-"},
	}
	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			id := NextID(nil, tt.category)
			if !strings.HasPrefix(id, tt.wantPrefix) {
				t.Errorf("NextID(%q) = %q, want prefix %q", tt.category, id, tt.wantPrefix)
			}
		})
	}
}

func TestNextIDConcurrent(t *testing.T) {
	// Parallel ID generation -- no duplicates
	ids := make(chan string, 1000)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ids <- NextID(nil, "Test")
			}
		}()
	}

	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := <-ids
		if seen[id] {
			t.Fatalf("duplicate concurrent ID: %s", id)
		}
		seen[id] = true
	}
}

func TestNextIDParallel(t *testing.T) {
	// 1000 concurrent goroutines generating IDs simultaneously
	const n = 1000
	results := make([]string, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = NextID(nil, "Parallel")
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool)
	for _, id := range results {
		if seen[id] {
			t.Fatalf("duplicate parallel ID: %s", id)
		}
		seen[id] = true
	}
}

func TestNextIDFormat(t *testing.T) {
	id := NextID(nil, "Stack")

	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		t.Fatalf("ID %q has %d parts, want 2 (PREFIX-SEQ)", id, len(parts))
	}

	prefix := parts[0]
	if prefix != "STACK" {
		t.Errorf("prefix = %q, want STACK", prefix)
	}

	seq := parts[1]
	if len(seq) < 3 {
		t.Errorf("seq %q has %d chars, want at least 3", seq, len(seq))
	}
}

func TestNextIDDifferentCategories(t *testing.T) {
	id1 := NextID(nil, "Stack")
	id2 := NextID(nil, "Data")

	if id1 == id2 {
		t.Errorf("IDs from different categories should differ: %s", id1)
	}

	if !strings.HasPrefix(id1, "STACK-") {
		t.Errorf("id1 prefix wrong: %s", id1)
	}
	if !strings.HasPrefix(id2, "DATA-") {
		t.Errorf("id2 prefix wrong: %s", id2)
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

func TestCategoryPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Stack", "STACK"},
		{"UI", "UI"},
		{"User Interface", "UI"},
		{"Authentication", "AUTH"},
		{"Data Model Design", "DMD"},
		{"A", "A"},
		{"Misc", "MISC"},
		{"API", "API"},
		{"Build & Deploy", "BD"},
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
