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
		{"escalation", "WHEN STUCK"},
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

// TestExecutePromptVariantFullHasBothLayers — the "full" variant must contain
// markers from both the guarded layer (rationalizations) and the escalation
// layer (when-stuck table). Otherwise the combination is broken.
func TestExecutePromptVariantFullHasBothLayers(t *testing.T) {
	t.Setenv("DEFER_EXEC_VARIANT", "full")
	got := ExecutePromptForVariant()
	if !strings.Contains(got, "COMMON RATIONALIZATIONS") {
		t.Error("full variant missing the guarded layer (rationalizations)")
	}
	if !strings.Contains(got, "WHEN STUCK") {
		t.Error("full variant missing the escalation layer (when-stuck table)")
	}
	if !strings.Contains(got, "CONCERNS:") {
		t.Error("full variant missing CONCERNS escalation status")
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
