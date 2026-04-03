package decision

import (
	"testing"
)

func makeDecisions() []Decision {
	ans := "Go with Gin"
	return []Decision{
		{ID: "STA-0001", Question: "Backend framework?", Impact: 9, Answer: &ans},
		{ID: "DAT-0002", Question: "Database?", DependsOn: []string{"STA-0001"}},
		{ID: "AUT-0003", Question: "Auth strategy?", DependsOn: []string{"STA-0001", "DAT-0002"}},
		{ID: "FEA-0005", Question: "Feature flags?", DependsOn: []string{"AUT-0003"}},
		{ID: "UII-0004", Question: "Frontend framework?"},
	}
}

func TestFindDependents(t *testing.T) {
	all := makeDecisions()

	deps := FindDependents("STA-0001", all)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of STA-0001, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	if !ids["DAT-0002"] {
		t.Error("expected DAT-0002 to depend on STA-0001")
	}
	if !ids["AUT-0003"] {
		t.Error("expected AUT-0003 to depend on STA-0001")
	}

	// UIX-0004 has no dependents
	deps = FindDependents("UII-0004", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependents of UIX-0004, got %d", len(deps))
	}

	// Non-existent ID
	deps = FindDependents("NOP-9999", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependents of NOP-9999, got %d", len(deps))
	}
}

func TestFindTransitiveDependents(t *testing.T) {
	all := makeDecisions()

	deps := FindTransitiveDependents("STA-0001", all)
	// STA-0001 -> DAT-0002, AUT-0003 (direct)
	// DAT-0002 -> AUT-0003 (already visited)
	// AUT-0003 -> FEA-0005
	if len(deps) != 3 {
		t.Fatalf("expected 3 transitive dependents of STA-0001, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	for _, expected := range []string{"DAT-0002", "AUT-0003", "FEA-0005"} {
		if !ids[expected] {
			t.Errorf("expected %s in transitive dependents of STA-0001", expected)
		}
	}

	// FEA-0005 has no dependents
	deps = FindTransitiveDependents("FEA-0005", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 transitive dependents of FEA-0005, got %d", len(deps))
	}
}

func TestFindDependencies(t *testing.T) {
	all := makeDecisions()

	// AUT-0003 depends on STA-0001 and DAT-0002
	deps := FindDependencies(all[2], all)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies for AUT-0003, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	if !ids["STA-0001"] {
		t.Error("expected STA-0001 as dependency of AUT-0003")
	}
	if !ids["DAT-0002"] {
		t.Error("expected DAT-0002 as dependency of AUT-0003")
	}

	// STA-0001 has no dependencies
	deps = FindDependencies(all[0], all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependencies for STA-0001, got %d", len(deps))
	}
}

func TestInvalidateDependent(t *testing.T) {
	ans := "some answer"
	d := Decision{
		ID:        "TES-0001",
		Answer:    &ans,
		Source:    "user",
		Delegated: true,
	}

	InvalidateDependent(&d)

	if d.Answer != nil {
		t.Error("expected Answer to be nil after invalidation")
	}
	if d.Source != "invalidated" {
		t.Errorf("expected Source = 'invalidated', got %q", d.Source)
	}
	if d.Delegated {
		t.Error("expected Delegated to be false after invalidation")
	}
}
