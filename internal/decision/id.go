package decision

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// categoryPrefix returns a 3-letter uppercase prefix for a category name.
//
// Rules:
//   - Single word: first 3 letters (pad with X if shorter)
//   - Multi-word: first letter of each word, take first 3 (pad from last word if fewer than 3 words)
//   - Empty/blank: "UNK"
func categoryPrefix(category string) string {
	words := splitWords(category)
	if len(words) == 0 {
		return "UNK"
	}
	if len(words) == 1 {
		w := strings.ToUpper(words[0])
		if len(w) >= 3 {
			return w[:3]
		}
		// Pad short words by repeating the last character: UI → UII, A → AAA
		for len(w) < 3 {
			w += string(w[len(w)-1])
		}
		return w[:3]
	}
	// Multi-word: first letter of each word
	var prefix string
	for _, w := range words {
		if len(w) > 0 {
			prefix += strings.ToUpper(w[:1])
		}
	}
	if len(prefix) >= 3 {
		return prefix[:3]
	}
	// Pad from last word
	lastWord := strings.ToUpper(words[len(words)-1])
	for len(prefix) < 3 && len(lastWord) > 1 {
		prefix += string(lastWord[1])
		lastWord = lastWord[1:]
	}
	return (prefix + "XXX")[:3]
}

// splitWords splits a string into words by spaces, hyphens, and underscores,
// filtering empty strings, and stripping non-alphanumeric characters.
func splitWords(s string) []string {
	// Replace hyphens and underscores with spaces
	s = strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return ' '
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			return r
		}
		return -1 // strip
	}, s)
	parts := strings.Fields(s)
	var words []string
	for _, p := range parts {
		if p != "" {
			words = append(words, p)
		}
	}
	return words
}

// NextID generates a unique decision ID like STA-0001, DAT-0002, etc.
// Stored without prefix. The UI adds @ when displaying/referencing.
func NextID(existing []Decision, category string) string {
	prefix := categoryPrefix(category)
	maxNum := 0
	for _, d := range existing {
		if strings.HasPrefix(d.ID, prefix+"-") {
			numStr := d.ID[len(prefix)+1:]
			if n, err := strconv.Atoi(numStr); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}
	return fmt.Sprintf("%s-%04d", prefix, maxNum+1)
}

// FeatureID generates a feature ID from a feature name.
// Stored without prefix. The UI adds # when displaying/referencing.
func FeatureID(name string) string {
	return categoryPrefix(name)
}
