package decision

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

// Counter to ensure uniqueness even within the same millisecond.
var idCounter uint64

// NextID generates a globally unique ID like STACK-0326220145-001.
// Uses category prefix + timestamp + atomic counter. No collisions possible.
func NextID(_ []Decision, category string) string {
	prefix := categoryPrefix(category)
	ts := time.Now().Format("0102150405") // MMDDHHMMSS (10 chars)
	seq := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%s-%03d", prefix, ts, seq%1000)
}

func categoryPrefix(category string) string {
	var clean strings.Builder
	for _, r := range category {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			clean.WriteRune(unicode.ToUpper(r))
		}
	}
	s := strings.TrimSpace(clean.String())

	if len(s) <= 6 && !strings.Contains(s, " ") {
		return s
	}

	words := strings.Fields(s)
	if len(words) > 1 {
		var initials strings.Builder
		for _, w := range words {
			if len(w) > 0 {
				initials.WriteByte(w[0])
			}
		}
		r := initials.String()
		if len(r) > 5 {
			return r[:5]
		}
		return r
	}

	if len(s) > 4 {
		return s[:4]
	}
	return s
}
