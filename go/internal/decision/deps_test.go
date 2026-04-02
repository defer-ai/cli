package decision

import (
	"testing"
)

func makeDecisions() []Decision {
	ans := "Go with Gin"
	return []Decision{
		{ID: "STACK-001", Question: "Backend framework?", Impact: 9, Answer: &ans},
		{ID: "DATA-002", Question: "Database?", DependsOn: []string{"STACK-001"}},
		{ID: "AUTH-003", Question: "Auth strategy?", DependsOn: []string{"STACK-001", "DATA-002"}},
		{ID: "FEAT-005", Question: "Feature flags?", DependsOn: []string{"AUTH-003"}},
		{ID: "UI-004", Question: "Frontend framework?"},
	}
}

func TestFindDependents(t *testing.T) {
	all := makeDecisions()

	deps := FindDependents("STACK-001", all)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of STACK-001, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	if !ids["DATA-002"] {
		t.Error("expected DATA-002 to depend on STACK-001")
	}
	if !ids["AUTH-003"] {
		t.Error("expected AUTH-003 to depend on STACK-001")
	}

	// UI-004 has no dependents
	deps = FindDependents("UI-004", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependents of UI-004, got %d", len(deps))
	}

	// Non-existent ID
	deps = FindDependents("NOPE-999", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependents of NOPE-999, got %d", len(deps))
	}
}

func TestFindTransitiveDependents(t *testing.T) {
	all := makeDecisions()

	deps := FindTransitiveDependents("STACK-001", all)
	// STACK-001 -> DATA-002, AUTH-003 (direct)
	// DATA-002 -> AUTH-003 (already visited)
	// AUTH-003 -> FEAT-005
	if len(deps) != 3 {
		t.Fatalf("expected 3 transitive dependents of STACK-001, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	for _, expected := range []string{"DATA-002", "AUTH-003", "FEAT-005"} {
		if !ids[expected] {
			t.Errorf("expected %s in transitive dependents of STACK-001", expected)
		}
	}

	// FEAT-005 has no dependents
	deps = FindTransitiveDependents("FEAT-005", all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 transitive dependents of FEAT-005, got %d", len(deps))
	}
}

func TestFindDependencies(t *testing.T) {
	all := makeDecisions()

	// AUTH-003 depends on STACK-001 and DATA-002
	deps := FindDependencies(all[2], all)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies for AUTH-003, got %d", len(deps))
	}
	ids := map[string]bool{}
	for _, d := range deps {
		ids[d.ID] = true
	}
	if !ids["STACK-001"] {
		t.Error("expected STACK-001 as dependency of AUTH-003")
	}
	if !ids["DATA-002"] {
		t.Error("expected DATA-002 as dependency of AUTH-003")
	}

	// STACK-001 has no dependencies
	deps = FindDependencies(all[0], all)
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependencies for STACK-001, got %d", len(deps))
	}
}

func TestInvalidateDependent(t *testing.T) {
	ans := "some answer"
	d := Decision{
		ID:        "TEST-001",
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
