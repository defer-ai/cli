package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// NotificationPriority determines display precedence and styling.
type NotificationPriority int

const (
	NotifyLow    NotificationPriority = iota // tool activity, progress
	NotifyMedium                             // auto-decided, domain status change
	NotifyHigh                               // build failed, decision changed, error
)

// Default auto-dismiss durations per priority level.
const (
	lowDuration    = 3 * time.Second
	mediumDuration = 5 * time.Second
	highDuration   = 10 * time.Second

	maxQueueSize = 10
)

// Notification represents a single message shown in the TUI status line.
type Notification struct {
	Text     string
	Priority NotificationPriority
	Time     time.Time
	Duration time.Duration // auto-dismiss after this; 0 = sticky until replaced
}

// expired reports whether the notification has outlived its duration.
// Sticky notifications (Duration == 0) never expire on their own.
func (n *Notification) expired(now time.Time) bool {
	if n.Duration == 0 {
		return false
	}
	return now.Sub(n.Time) >= n.Duration
}

// NotificationManager maintains an active notification and a bounded queue.
type NotificationManager struct {
	current *Notification
	queue   []Notification
}

// NewNotificationManager creates a ready-to-use manager.
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{}
}

// Push adds a notification. Higher priority replaces the current notification
// immediately; same priority also replaces (newer wins). Lower priority is
// queued. The queue holds at most maxQueueSize entries; oldest are dropped.
func (nm *NotificationManager) Push(text string, priority NotificationPriority, duration time.Duration) {
	// Apply default durations when none specified.
	if duration < 0 {
		duration = 0
	}
	if duration == 0 {
		switch priority {
		case NotifyLow:
			duration = lowDuration
		case NotifyMedium:
			duration = mediumDuration
		case NotifyHigh:
			// High with Duration=0 stays sticky.
			// Callers must pass 0 explicitly for sticky; we leave it.
		}
	}

	n := Notification{
		Text:     text,
		Priority: priority,
		Time:     time.Now(),
		Duration: duration,
	}

	if nm.current == nil || priority >= nm.current.Priority {
		// Demote current to queue if it hasn't expired.
		if nm.current != nil && !nm.current.expired(time.Now()) {
			nm.enqueue(*nm.current)
		}
		nm.current = &n
		return
	}

	// Lower priority: queue it.
	nm.enqueue(n)
}

// enqueue appends to the queue, evicting the oldest entry when full.
func (nm *NotificationManager) enqueue(n Notification) {
	if len(nm.queue) >= maxQueueSize {
		nm.queue = nm.queue[1:]
	}
	nm.queue = append(nm.queue, n)
}

// Tick should be called periodically (every ~100ms). It auto-dismisses
// expired notifications and promotes the next queued notification.
func (nm *NotificationManager) Tick() {
	now := time.Now()

	if nm.current != nil && nm.current.expired(now) {
		nm.current = nil
	}

	// Prune expired entries from the queue.
	alive := nm.queue[:0]
	for _, n := range nm.queue {
		if !n.expired(now) {
			alive = append(alive, n)
		}
	}
	nm.queue = alive

	// Promote highest-priority queued notification if current is empty.
	if nm.current == nil && len(nm.queue) > 0 {
		best := 0
		for i := 1; i < len(nm.queue); i++ {
			if nm.queue[i].Priority > nm.queue[best].Priority {
				best = i
			} else if nm.queue[i].Priority == nm.queue[best].Priority &&
				nm.queue[i].Time.After(nm.queue[best].Time) {
				best = i
			}
		}
		promoted := nm.queue[best]
		nm.current = &promoted
		nm.queue = append(nm.queue[:best], nm.queue[best+1:]...)
	}
}

// Current returns the active notification, or nil if nothing to show.
func (nm *NotificationManager) Current() *Notification {
	return nm.current
}

// Render returns a single styled line for the TUI. If there is no active
// notification the empty string is returned.
func (nm *NotificationManager) Render(width int) string {
	if nm.current == nil {
		return ""
	}

	icon, style := iconAndStyle(nm.current.Priority)
	text := nm.current.Text
	maxText := width - lipgloss.Width(icon) - 1 // icon + space
	if maxText < 4 {
		maxText = 4
	}
	if lipgloss.Width(text) > maxText {
		text = trunc(text, maxText)
	}

	return style.Render(icon + " " + text)
}

// iconAndStyle maps a priority to its display icon and lipgloss style.
func iconAndStyle(p NotificationPriority) (string, lipgloss.Style) {
	switch p {
	case NotifyHigh:
		return "●", RedStyle.Bold(true)
	case NotifyMedium:
		return "◆", YellowStyle
	default:
		return "○", DimStyle
	}
}
