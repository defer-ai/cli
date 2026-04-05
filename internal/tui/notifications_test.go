package tui

import (
	"strings"
	"testing"
	"time"
)

func TestPushReplacesLowerPriority(t *testing.T) {
	nm := NewNotificationManager()

	nm.Push("low message", NotifyLow, 3*time.Second)
	if nm.Current() == nil || nm.Current().Text != "low message" {
		t.Fatal("expected low message as current")
	}

	// Medium should replace low.
	nm.Push("medium message", NotifyMedium, 5*time.Second)
	if nm.Current().Text != "medium message" {
		t.Errorf("current = %q, want %q", nm.Current().Text, "medium message")
	}
	if nm.Current().Priority != NotifyMedium {
		t.Errorf("priority = %d, want NotifyMedium", nm.Current().Priority)
	}

	// High should replace medium.
	nm.Push("high message", NotifyHigh, 10*time.Second)
	if nm.Current().Text != "high message" {
		t.Errorf("current = %q, want %q", nm.Current().Text, "high message")
	}
	if nm.Current().Priority != NotifyHigh {
		t.Errorf("priority = %d, want NotifyHigh", nm.Current().Priority)
	}

	// The demoted medium should be in the queue.
	found := false
	for _, n := range nm.queue {
		if n.Text == "medium message" {
			found = true
		}
	}
	if !found {
		t.Error("medium message should be in queue after being replaced by high")
	}
}

func TestSamePriorityNewerReplaces(t *testing.T) {
	nm := NewNotificationManager()

	nm.Push("first medium", NotifyMedium, 5*time.Second)
	nm.Push("second medium", NotifyMedium, 5*time.Second)

	if nm.Current().Text != "second medium" {
		t.Errorf("current = %q, want %q", nm.Current().Text, "second medium")
	}
}

func TestLowerPriorityQueued(t *testing.T) {
	nm := NewNotificationManager()

	nm.Push("high first", NotifyHigh, 10*time.Second)
	nm.Push("low after", NotifyLow, 3*time.Second)

	// Current should still be high.
	if nm.Current().Text != "high first" {
		t.Errorf("current = %q, want %q", nm.Current().Text, "high first")
	}

	// Low should be queued.
	if len(nm.queue) != 1 || nm.queue[0].Text != "low after" {
		t.Errorf("queue = %v, want [low after]", nm.queue)
	}
}

func TestAutoDismissAfterDuration(t *testing.T) {
	nm := NewNotificationManager()

	// Inject a notification that already expired.
	past := time.Now().Add(-4 * time.Second)
	nm.current = &Notification{
		Text:     "should expire",
		Priority: NotifyLow,
		Time:     past,
		Duration: 3 * time.Second,
	}

	nm.Tick()

	if nm.Current() != nil {
		t.Errorf("expected nil after expiry, got %q", nm.Current().Text)
	}
}

func TestAutoDismissPromotesFromQueue(t *testing.T) {
	nm := NewNotificationManager()

	// Current is expired, queue has a live notification.
	past := time.Now().Add(-4 * time.Second)
	nm.current = &Notification{
		Text:     "expired",
		Priority: NotifyLow,
		Time:     past,
		Duration: 3 * time.Second,
	}
	nm.queue = []Notification{
		{Text: "queued medium", Priority: NotifyMedium, Time: time.Now(), Duration: 5 * time.Second},
	}

	nm.Tick()

	if nm.Current() == nil || nm.Current().Text != "queued medium" {
		t.Errorf("expected promoted queued medium, got %v", nm.Current())
	}
	if len(nm.queue) != 0 {
		t.Errorf("queue should be empty after promotion, has %d", len(nm.queue))
	}
}

func TestHighPrioritySticky(t *testing.T) {
	nm := NewNotificationManager()

	// Push high with Duration=0 (sticky).
	nm.Push("sticky error", NotifyHigh, 0)

	if nm.Current() == nil || nm.Current().Text != "sticky error" {
		t.Fatal("expected sticky error as current")
	}
	if nm.Current().Duration != 0 {
		t.Errorf("duration = %v, want 0 (sticky)", nm.Current().Duration)
	}

	// Simulate many ticks -- should not expire.
	// Pretend 30 seconds have passed by backdating the notification.
	nm.current.Time = time.Now().Add(-30 * time.Second)
	nm.Tick()

	if nm.Current() == nil || nm.Current().Text != "sticky error" {
		t.Error("sticky notification should not be auto-dismissed")
	}
}

func TestQueueOrdering(t *testing.T) {
	nm := NewNotificationManager()

	// Fill queue behind a high-priority current.
	nm.Push("high", NotifyHigh, 10*time.Second)
	nm.Push("low 1", NotifyLow, 3*time.Second)
	nm.Push("medium 1", NotifyMedium, 5*time.Second)
	nm.Push("low 2", NotifyLow, 3*time.Second)

	if nm.Current().Text != "high" {
		t.Fatalf("current = %q, want high", nm.Current().Text)
	}

	// Expire the current to force promotion.
	nm.current.Time = time.Now().Add(-11 * time.Second)
	nm.current.Duration = 10 * time.Second
	nm.Tick()

	// The medium should be promoted (highest priority in queue).
	if nm.Current() == nil {
		t.Fatal("expected promotion from queue")
	}
	if nm.Current().Text != "medium 1" {
		t.Errorf("promoted = %q, want medium 1 (highest priority in queue)", nm.Current().Text)
	}
}

func TestQueueMaxSize(t *testing.T) {
	nm := NewNotificationManager()

	// Set a high current so everything else queues.
	nm.Push("current", NotifyHigh, 10*time.Second)

	// Push 12 low-priority notifications.
	for i := 0; i < 12; i++ {
		nm.Push("low", NotifyLow, 3*time.Second)
	}

	if len(nm.queue) > maxQueueSize {
		t.Errorf("queue size = %d, want <= %d", len(nm.queue), maxQueueSize)
	}
}

func TestDefaultDurations(t *testing.T) {
	nm := NewNotificationManager()

	// Push with duration=0 triggers default for Low and Medium.
	// For High, duration=0 means sticky.
	nm.Push("low", NotifyLow, 0)
	if nm.Current().Duration != lowDuration {
		t.Errorf("low default duration = %v, want %v", nm.Current().Duration, lowDuration)
	}

	nm.Push("medium", NotifyMedium, 0)
	if nm.Current().Duration != mediumDuration {
		t.Errorf("medium default duration = %v, want %v", nm.Current().Duration, mediumDuration)
	}

	nm.Push("high sticky", NotifyHigh, 0)
	if nm.Current().Duration != 0 {
		t.Errorf("high sticky duration = %v, want 0", nm.Current().Duration)
	}
}

func TestRenderEmpty(t *testing.T) {
	nm := NewNotificationManager()
	out := nm.Render(80)
	if out != "" {
		t.Errorf("render with no notification = %q, want empty", out)
	}
}

func TestRenderLowPriority(t *testing.T) {
	nm := NewNotificationManager()
	nm.Push("compiling...", NotifyLow, 3*time.Second)

	out := nm.Render(80)
	if out == "" {
		t.Fatal("expected non-empty render")
	}
	if !strings.Contains(out, "compiling...") {
		t.Errorf("render = %q, missing notification text", out)
	}
	if !strings.Contains(out, "○") {
		t.Errorf("render = %q, missing low-priority icon", out)
	}
}

func TestRenderMediumPriority(t *testing.T) {
	nm := NewNotificationManager()
	nm.Push("auto-decided Stack", NotifyMedium, 5*time.Second)

	out := nm.Render(80)
	if !strings.Contains(out, "◆") {
		t.Errorf("render = %q, missing medium-priority icon", out)
	}
	if !strings.Contains(out, "auto-decided Stack") {
		t.Errorf("render = %q, missing notification text", out)
	}
}

func TestRenderHighPriority(t *testing.T) {
	nm := NewNotificationManager()
	nm.Push("build failed", NotifyHigh, 10*time.Second)

	out := nm.Render(80)
	if !strings.Contains(out, "●") {
		t.Errorf("render = %q, missing high-priority icon", out)
	}
	if !strings.Contains(out, "build failed") {
		t.Errorf("render = %q, missing notification text", out)
	}
}

func TestRenderTruncatesLongText(t *testing.T) {
	nm := NewNotificationManager()
	long := strings.Repeat("x", 200)
	nm.Push(long, NotifyLow, 3*time.Second)

	out := nm.Render(40)
	// The rendered output should fit within roughly 40 columns.
	// We just check it's shorter than the full text.
	if strings.Contains(out, long) {
		t.Error("render should truncate long text")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated text should end with ellipsis")
	}
}

func TestTickPrunesExpiredQueue(t *testing.T) {
	nm := NewNotificationManager()

	past := time.Now().Add(-10 * time.Second)
	nm.current = &Notification{
		Text: "alive", Priority: NotifyHigh, Time: time.Now(), Duration: 0, // sticky
	}
	nm.queue = []Notification{
		{Text: "expired1", Priority: NotifyLow, Time: past, Duration: 3 * time.Second},
		{Text: "alive1", Priority: NotifyMedium, Time: time.Now(), Duration: 5 * time.Second},
		{Text: "expired2", Priority: NotifyLow, Time: past, Duration: 3 * time.Second},
	}

	nm.Tick()

	// Only alive1 should remain in queue.
	if len(nm.queue) != 1 {
		t.Errorf("queue size = %d, want 1", len(nm.queue))
	}
	if nm.queue[0].Text != "alive1" {
		t.Errorf("queue[0] = %q, want alive1", nm.queue[0].Text)
	}
}
