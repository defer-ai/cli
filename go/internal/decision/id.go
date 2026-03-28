package decision

import (
	"fmt"
	"strings"
	"sync/atomic"
	"unicode"
)

// Counter to ensure uniqueness within a session.
var idCounter uint64

// NextID generates a unique ID like STACK-001, STACK-002, etc.
// Uses category prefix + atomic counter. No timestamp, no collision possible within a process.
func NextID(_ []Decision, category string) string {
	prefix := categoryPrefix(category)
	seq := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%03d", prefix, seq)
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
