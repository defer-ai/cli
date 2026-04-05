package decision

import "testing"

func TestTrailers_MixedDecisions(t *testing.T) {
	answered := "Go"
	store := &DecisionStore{
		Task: "test",
		Decisions: []Decision{
			{ID: "STA-0001", Answer: &answered},
			{ID: "STA-0002", Answer: nil}, // pending
			{ID: "DAT-0001", Answer: &answered},
		},
	}

	got := Trailers(store)
	want := "Decision-Ref: @STA-0001\nDecision-Ref: @DAT-0001"
	if got != want {
		t.Errorf("Trailers() =\n%q\nwant\n%q", got, want)
	}
}

func TestTrailers_EmptyStore(t *testing.T) {
	store := &DecisionStore{Task: "test", Decisions: []Decision{}}
	got := Trailers(store)
	if got != "" {
		t.Errorf("Trailers() = %q, want empty", got)
	}
}

func TestTrailers_NilStore(t *testing.T) {
	got := Trailers(nil)
	if got != "" {
		t.Errorf("Trailers(nil) = %q, want empty", got)
	}
}

func TestTrailersForIDs_Filtering(t *testing.T) {
	answered := "yes"
	store := &DecisionStore{
		Task: "test",
		Decisions: []Decision{
			{ID: "STA-0001", Answer: &answered},
			{ID: "STA-0002", Answer: &answered},
			{ID: "DAT-0001", Answer: &answered},
		},
	}

	got := TrailersForIDs(store, []string{"STA-0001", "DAT-0001"})
	want := "Decision-Ref: @STA-0001\nDecision-Ref: @DAT-0001"
	if got != want {
		t.Errorf("TrailersForIDs() =\n%q\nwant\n%q", got, want)
	}
}

func TestTrailersForIDs_SkipsPending(t *testing.T) {
	answered := "yes"
	store := &DecisionStore{
		Task: "test",
		Decisions: []Decision{
			{ID: "STA-0001", Answer: &answered},
			{ID: "STA-0002", Answer: nil}, // pending — requested but not answered
		},
	}

	got := TrailersForIDs(store, []string{"STA-0001", "STA-0002"})
	want := "Decision-Ref: @STA-0001"
	if got != want {
		t.Errorf("TrailersForIDs() = %q, want %q", got, want)
	}
}

func TestTrailersForIDs_EmptyIDs(t *testing.T) {
	answered := "yes"
	store := &DecisionStore{
		Task: "test",
		Decisions: []Decision{
			{ID: "STA-0001", Answer: &answered},
		},
	}

	got := TrailersForIDs(store, []string{})
	if got != "" {
		t.Errorf("TrailersForIDs(empty) = %q, want empty", got)
	}
}

func TestTrailersForIDs_NilStore(t *testing.T) {
	got := TrailersForIDs(nil, []string{"STA-0001"})
	if got != "" {
		t.Errorf("TrailersForIDs(nil) = %q, want empty", got)
	}
}
