package decision

import (
	"testing"
)

func ptr(s string) *string { return &s }

func TestComputeStats_EmptyStore(t *testing.T) {
	store := &DecisionStore{Decisions: []Decision{}}
	s := ComputeStats(store)

	if s.Total != 0 {
		t.Errorf("expected Total=0, got %d", s.Total)
	}
	if s.Answered != 0 {
		t.Errorf("expected Answered=0, got %d", s.Answered)
	}
	if s.Pending != 0 {
		t.Errorf("expected Pending=0, got %d", s.Pending)
	}
	if s.MaxDepth != 0 {
		t.Errorf("expected MaxDepth=0, got %d", s.MaxDepth)
	}
	if len(s.MostRevised) != 0 {
		t.Errorf("expected no MostRevised, got %d", len(s.MostRevised))
	}
}

func TestComputeStats_NilStore(t *testing.T) {
	s := ComputeStats(nil)
	if s.Total != 0 {
		t.Errorf("expected Total=0 for nil store, got %d", s.Total)
	}
}

func TestComputeStats_AutoUserMix(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "A-0001", Answer: ptr("yes"), Source: "auto", OriginalSource: "auto", Category: "Stack"},
			{ID: "A-0002", Answer: ptr("no"), Source: "auto", OriginalSource: "auto", Category: "Stack"},
			{ID: "A-0003", Answer: ptr("maybe"), Source: "user", OriginalSource: "user", Category: "Data"},
			{ID: "A-0004", Category: "Data"}, // pending
		},
	}
	s := ComputeStats(store)

	if s.Total != 4 {
		t.Errorf("expected Total=4, got %d", s.Total)
	}
	if s.Answered != 3 {
		t.Errorf("expected Answered=3, got %d", s.Answered)
	}
	if s.Pending != 1 {
		t.Errorf("expected Pending=1, got %d", s.Pending)
	}
	if s.AutoCount != 2 {
		t.Errorf("expected AutoCount=2, got %d", s.AutoCount)
	}
	if s.UserCount != 1 {
		t.Errorf("expected UserCount=1, got %d", s.UserCount)
	}
}

func TestComputeStats_OverrideDetection(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			// Originally auto, user overrode
			{ID: "O-0001", Answer: ptr("v2"), Source: "user", OriginalSource: "auto", RevisionCount: 1},
			// Originally auto, still auto
			{ID: "O-0002", Answer: ptr("v1"), Source: "auto", OriginalSource: "auto"},
			// Originally auto, user overrode
			{ID: "O-0003", Answer: ptr("v3"), Source: "user", OriginalSource: "auto", RevisionCount: 2},
			// Originally user, still user (not an override)
			{ID: "O-0004", Answer: ptr("v1"), Source: "user", OriginalSource: "user"},
		},
	}
	s := ComputeStats(store)

	if s.OverrideTotal != 3 {
		t.Errorf("expected OverrideTotal=3 (all originally auto), got %d", s.OverrideTotal)
	}
	if s.OverrideCount != 2 {
		t.Errorf("expected OverrideCount=2, got %d", s.OverrideCount)
	}
}

func TestComputeStats_CategoryGrouping(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "C-0001", Category: "Stack", Answer: ptr("Go")},
			{ID: "C-0002", Category: "Stack", Answer: ptr("Cobra")},
			{ID: "C-0003", Category: "Data", Answer: ptr("Postgres")},
			{ID: "C-0004", Category: "Data"},   // pending
			{ID: "C-0005", Category: ""},        // no category
		},
	}
	s := ComputeStats(store)

	stack, ok := s.ByCategory["Stack"]
	if !ok {
		t.Fatal("expected Stack category")
	}
	if stack.Total != 2 || stack.Answered != 2 || stack.Pending != 0 {
		t.Errorf("Stack: expected 2/2/0, got %d/%d/%d", stack.Total, stack.Answered, stack.Pending)
	}

	data, ok := s.ByCategory["Data"]
	if !ok {
		t.Fatal("expected Data category")
	}
	if data.Total != 2 || data.Answered != 1 || data.Pending != 1 {
		t.Errorf("Data: expected 2/1/1, got %d/%d/%d", data.Total, data.Answered, data.Pending)
	}

	none, ok := s.ByCategory["(none)"]
	if !ok {
		t.Fatal("expected (none) category for empty category")
	}
	if none.Total != 1 {
		t.Errorf("(none): expected 1, got %d", none.Total)
	}
}

func TestComputeStats_ImpactDistribution(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "I-0001", Impact: 10}, // high
			{ID: "I-0002", Impact: 7},  // high
			{ID: "I-0003", Impact: 6},  // medium
			{ID: "I-0004", Impact: 4},  // medium
			{ID: "I-0005", Impact: 3},  // low
			{ID: "I-0006", Impact: 0},  // low
		},
	}
	s := ComputeStats(store)

	if s.ImpactHigh != 2 {
		t.Errorf("expected ImpactHigh=2, got %d", s.ImpactHigh)
	}
	if s.ImpactMedium != 2 {
		t.Errorf("expected ImpactMedium=2, got %d", s.ImpactMedium)
	}
	if s.ImpactLow != 2 {
		t.Errorf("expected ImpactLow=2, got %d", s.ImpactLow)
	}
}

func TestComputeStats_DependencyDepth(t *testing.T) {
	// Chain: A -> B -> C -> D (depth 3)
	// E is isolated (depth 0)
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "A-0001"},
			{ID: "B-0001", DependsOn: []string{"A-0001"}},
			{ID: "C-0001", DependsOn: []string{"B-0001"}},
			{ID: "D-0001", DependsOn: []string{"C-0001"}},
			{ID: "E-0001"},
		},
	}
	s := ComputeStats(store)

	if s.MaxDepth != 3 {
		t.Errorf("expected MaxDepth=3, got %d", s.MaxDepth)
	}
	if s.MaxDepthChain == "" {
		t.Error("expected non-empty MaxDepthChain")
	}
}

func TestComputeStats_DependencyDepthBranching(t *testing.T) {
	// A -> B -> D (depth 2)
	// A -> C     (depth 1)
	// max depth should be 2
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "A-0001"},
			{ID: "B-0001", DependsOn: []string{"A-0001"}},
			{ID: "C-0001", DependsOn: []string{"A-0001"}},
			{ID: "D-0001", DependsOn: []string{"B-0001"}},
		},
	}
	s := ComputeStats(store)

	if s.MaxDepth != 2 {
		t.Errorf("expected MaxDepth=2, got %d", s.MaxDepth)
	}
}

func TestComputeStats_MostRevised(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "R-0001", Question: "Q1", RevisionCount: 5, Answer: ptr("a")},
			{ID: "R-0002", Question: "Q2", RevisionCount: 0, Answer: ptr("b")},
			{ID: "R-0003", Question: "Q3", RevisionCount: 3, Answer: ptr("c")},
			{ID: "R-0004", Question: "Q4", RevisionCount: 1, Answer: ptr("d")},
			{ID: "R-0005", Question: "Q5", RevisionCount: 7, Answer: ptr("e")},
			{ID: "R-0006", Question: "Q6", RevisionCount: 2, Answer: ptr("f")},
		},
	}
	s := ComputeStats(store)

	if len(s.MostRevised) != 3 {
		t.Fatalf("expected 3 most revised, got %d", len(s.MostRevised))
	}

	// Should be sorted descending: R-0005 (7), R-0001 (5), R-0003 (3)
	if s.MostRevised[0].ID != "R-0005" || s.MostRevised[0].RevisionCount != 7 {
		t.Errorf("expected first=%s(7), got %s(%d)", "R-0005", s.MostRevised[0].ID, s.MostRevised[0].RevisionCount)
	}
	if s.MostRevised[1].ID != "R-0001" || s.MostRevised[1].RevisionCount != 5 {
		t.Errorf("expected second=%s(5), got %s(%d)", "R-0001", s.MostRevised[1].ID, s.MostRevised[1].RevisionCount)
	}
	if s.MostRevised[2].ID != "R-0003" || s.MostRevised[2].RevisionCount != 3 {
		t.Errorf("expected third=%s(3), got %s(%d)", "R-0003", s.MostRevised[2].ID, s.MostRevised[2].RevisionCount)
	}
}

func TestComputeStats_MostRevisedSkipsZero(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "Z-0001", RevisionCount: 0},
			{ID: "Z-0002", RevisionCount: 0},
		},
	}
	s := ComputeStats(store)

	if len(s.MostRevised) != 0 {
		t.Errorf("expected 0 most revised when all have 0 revisions, got %d", len(s.MostRevised))
	}
}

func TestComputeStats_MostRevisedFewerThanThree(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "F-0001", RevisionCount: 2, Question: "Q1"},
		},
	}
	s := ComputeStats(store)

	if len(s.MostRevised) != 1 {
		t.Errorf("expected 1 most revised, got %d", len(s.MostRevised))
	}
}

func TestComputeStats_NoDependencies(t *testing.T) {
	store := &DecisionStore{
		Decisions: []Decision{
			{ID: "N-0001"},
			{ID: "N-0002"},
		},
	}
	s := ComputeStats(store)

	if s.MaxDepth != 0 {
		t.Errorf("expected MaxDepth=0 with no dependencies, got %d", s.MaxDepth)
	}
}
