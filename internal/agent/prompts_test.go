package agent

import (
	"strings"
	"testing"
)

// TestExecutePromptForVariantDefault ensures that with no env var the
// dispatcher returns the current default skill-based prompt unchanged.
// This is the regression guard: adding new variants must never silently
// change runtime behavior.
func TestExecutePromptForVariantDefault(t *testing.T) {
	t.Setenv("DEFER_EXEC_VARIANT", "")
	got := ExecutePromptForVariant()
	if got != ExecutePromptTemplate {
		t.Error("default ExecutePromptForVariant() should return ExecutePromptTemplate")
	}
	if !strings.Contains(got, "narrates each choice") {
		t.Error("default execute prompt should contain the architect-narrator role frame")
	}
}

func TestExecutePromptForVariantSelection(t *testing.T) {
	cases := []struct {
		env  string
		want string // a marker substring unique to that variant
	}{
		{"rules", "CRITICAL RULES"},
		{"anchor", "PROTOCOL — non-negotiable"},
		{"guarded", "COMMON RATIONALIZATIONS"},
		{"full", "COMMON RATIONALIZATIONS"},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("DEFER_EXEC_VARIANT", tc.env)
			got := ExecutePromptForVariant()
			if !strings.Contains(got, tc.want) {
				t.Errorf("variant %q should contain %q, got prompt of length %d", tc.env, tc.want, len(got))
			}
		})
	}
}

// TestEscalationDeprecated — after the 3×4 benchmark showed the escalation
// variant suppresses total decision count by 3x, it was removed from the
// dispatcher and routes to the default prompt instead. This guards the
// deprecation so the harmful variant can't accidentally be reactivated.
func TestEscalationDeprecated(t *testing.T) {
	t.Setenv("DEFER_EXEC_VARIANT", "escalation")
	if ExecutePromptForVariant() != ExecutePromptTemplate {
		t.Error("escalation variant should fall back to default (deprecated)")
	}
}

// TestExecutePromptVariantFullHasBothLayers — the "full" variant must contain
// markers from both the guarded layer (rationalizations + red flags) and the
// anchor layer (tool-anchored protocol). Otherwise the combination is broken.
// Previously "full" was guarded+escalation; rebuilt to guarded+anchor after
// escalation tested as actively harmful.
func TestExecutePromptVariantFullHasBothLayers(t *testing.T) {
	t.Setenv("DEFER_EXEC_VARIANT", "full")
	got := ExecutePromptForVariant()
	if !strings.Contains(got, "COMMON RATIONALIZATIONS") {
		t.Error("full variant missing the guarded layer (rationalizations)")
	}
	if !strings.Contains(got, "RED FLAGS") {
		t.Error("full variant missing the guarded layer (red flags)")
	}
	if !strings.Contains(got, "PROTOCOL — non-negotiable") {
		t.Error("full variant missing the anchor layer (tool-anchored protocol)")
	}
	if !strings.Contains(got, "After EVERY Write or Edit tool result") {
		t.Error("full variant missing the anchor layer's tool-boundary requirement")
	}
}

func TestVerifyPromptForVariantDefault(t *testing.T) {
	t.Setenv("DEFER_VERIFY_VARIANT", "")
	if VerifyPromptForVariant() != VerifyPrompt {
		t.Error("default VerifyPromptForVariant() should return VerifyPrompt")
	}
}

func TestVerifyPromptForVariantCeremony(t *testing.T) {
	t.Setenv("DEFER_VERIFY_VARIANT", "ceremony")
	got := VerifyPromptForVariant()
	for _, marker := range []string{"IDENTIFY", "RUN", "READ", "VERIFY", "CLAIM", "show your work"} {
		if !strings.Contains(got, marker) {
			t.Errorf("ceremony variant missing %q gate step", marker)
		}
	}
	if !strings.Contains(got, "VERIFIED OK") {
		t.Error("ceremony variant must still produce a VERIFIED OK / NEEDS FIX verdict")
	}
}

func TestExtractPromptForVariantDefault(t *testing.T) {
	t.Setenv("DEFER_EXTRACT_VARIANT", "")
	if ExtractPromptForVariant() != ExtractPrompt {
		t.Error("default ExtractPromptForVariant() should return ExtractPrompt")
	}
}

func TestExtractPromptForVariantCoverage(t *testing.T) {
	t.Setenv("DEFER_EXTRACT_VARIANT", "coverage")
	got := ExtractPromptForVariant()
	for _, marker := range []string{`"decisions"`, `"coverage"`, "implemented_in", "decision_id"} {
		if !strings.Contains(got, marker) {
			t.Errorf("coverage variant missing %q field", marker)
		}
	}
}

// TestUnknownVariantsFallBackToDefault — defensive guard: a typo in the env
// var should not break behavior, it should silently use the default.
func TestUnknownVariantsFallBackToDefault(t *testing.T) {
	t.Setenv("DEFER_EXEC_VARIANT", "nope-not-a-real-variant")
	if ExecutePromptForVariant() != ExecutePromptTemplate {
		t.Error("unknown DEFER_EXEC_VARIANT should fall back to default")
	}
	t.Setenv("DEFER_VERIFY_VARIANT", "also-nonsense")
	if VerifyPromptForVariant() != VerifyPrompt {
		t.Error("unknown DEFER_VERIFY_VARIANT should fall back to default")
	}
	t.Setenv("DEFER_EXTRACT_VARIANT", "still-nonsense")
	if ExtractPromptForVariant() != ExtractPrompt {
		t.Error("unknown DEFER_EXTRACT_VARIANT should fall back to default")
	}
}
