package tui

import (
	"strings"
)

// MascotMood determines which expression to render.
type MascotMood int

const (
	MoodIdle   MascotMood = iota
	MoodActive            // thinking, executing, planning, verifying — all use this
	MoodDone
	MoodError
)

func StatusToMood(status string) MascotMood {
	switch status {
	case "thinking", "decomposing", "executing", "planning", "verifying", "waiting":
		return MoodActive
	case "done":
		return MoodDone
	case "error":
		return MoodError
	default:
		return MoodIdle
	}
}

// noiseRune returns ▓ or ░ based on position and tick for shimmer effect.
func noiseRune(col, row, tick int) rune {
	// Hash position + tick so the pattern changes each tick
	h := uint32(col)*2654435761 + uint32(row)*2246822519 + uint32(tick)*3266489917
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	if h%2 == 0 {
		return '▓'
	}
	return '░'
}

// RenderMascot returns a multi-line string of the mascot at the given tick.
// The mascot is two box-drawn eyes, 4 lines tall, ~17 chars wide.
//
//	Idle (shimmer noise):
//	 ╭────╮ ╭────╮
//	 │▓░▓░│ │░▓░▓│
//	 │░▓░▓│ │▓░▓░│
//	 ╰────╯ ╰────╯
//
//	Active (horizontal lines):
//	 ╭────╮ ╭────╮
//	 │────│ │────│
//	 ╰────╯ ╰────╯
//
//	Done (happy curves):
//	 ╭────╮ ╭────╮
//	 │‿‿‿‿│ │‿‿‿‿│
//	 ╰────╯ ╰────╯
//
//	Error (X eyes):
//	 ╭────╮ ╭────╮
//	 │╳╳╳╳│ │╳╳╳╳│
//	 ╰────╯ ╰────╯
func RenderMascot(mood MascotMood, tick int) string {
	const (
		innerW = 4 // chars inside the box walls
		gap    = " " // between the two eyes
	)

	top := " ╭" + strings.Repeat("─", innerW) + "╮" + gap + "╭" + strings.Repeat("─", innerW) + "╮"
	bot := " ╰" + strings.Repeat("─", innerW) + "╯" + gap + "╰" + strings.Repeat("─", innerW) + "╯"

	var lines []string

	switch mood {
	case MoodIdle:
		// Two rows of noise fill, shimmer changes each tick
		row1L := buildNoiseLine(innerW, 0, tick, 0)
		row2L := buildNoiseLine(innerW, 1, tick, 0)
		row1R := buildNoiseLine(innerW, 0, tick, 7) // offset seed for right eye
		row2R := buildNoiseLine(innerW, 1, tick, 7)

		lines = []string{
			top,
			" │" + row1L + "│" + gap + "│" + row1R + "│",
			" │" + row2L + "│" + gap + "│" + row2R + "│",
			bot,
		}

	case MoodActive:
		// Squinting: single row of horizontal dashes
		fill := strings.Repeat("─", innerW)
		lines = []string{
			top,
			" │" + fill + "│" + gap + "│" + fill + "│",
			bot,
		}

	case MoodDone:
		// Happy: single row of curves
		fill := strings.Repeat("‿", innerW)
		lines = []string{
			top,
			" │" + fill + "│" + gap + "│" + fill + "│",
			bot,
		}

	case MoodError:
		// Error: single row of X marks
		fill := strings.Repeat("╳", innerW)
		lines = []string{
			top,
			" │" + fill + "│" + gap + "│" + fill + "│",
			bot,
		}
	}

	// Apply accent color to each line
	var result []string
	for _, line := range lines {
		result = append(result, AccentStyle.Render(line))
	}
	return strings.Join(result, "\n")
}

// buildNoiseLine creates a string of innerW noise runes for a given row/tick/seed.
func buildNoiseLine(width, row, tick, seed int) string {
	var sb strings.Builder
	for col := 0; col < width; col++ {
		sb.WriteRune(noiseRune(col+seed, row, tick))
	}
	return sb.String()
}
