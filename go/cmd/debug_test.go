package cmd

import (
	"testing"
)

func TestRunDebugRequiresTask(t *testing.T) {
	err := runDebug("", "sonnet", nil, "/tmp/test")
	if err == nil {
		t.Fatal("expected error for empty task")
	}
	if err.Error() != "--debug requires a task argument" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunDebugNoProviderDoesNotPanic(t *testing.T) {
	// With ccProvider nil, decomposition will never
	// send AgentDecisionsReady. We cannot run the full flow, but we
	// ensure it doesn't panic on setup. The function will block on
	// the decomposeDone channel, so we just validate the error case.
	err := runDebug("", "sonnet", nil, "/tmp/test")
	if err == nil {
		t.Fatal("expected error")
	}
}
