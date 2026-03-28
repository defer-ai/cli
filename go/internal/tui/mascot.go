package tui

import (
	"math"
	"strings"
)

// MascotMood determines which expression to render.
type MascotMood int

const (
	MoodIdle MascotMood = iota
	MoodThinking
	MoodExecuting
	MoodDone
	MoodError
)

func StatusToMood(status string) MascotMood {
	switch status {
	case "thinking", "decomposing":
		return MoodThinking
	case "executing", "planning", "verifying":
		return MoodExecuting
	case "done":
		return MoodDone
	case "error":
		return MoodError
	default:
		return MoodIdle
	}
}

// RenderMascot returns a multi-line string of the mascot at the given tick.
// Simplified version: circle eyes with noise fill and half-block characters.
func RenderMascot(mood MascotMood, tick int) string {
	const (
		srcSize = 30
		scale   = 2
		dspSize = srcSize / scale
		gap     = 4
	)

	rows := (dspSize + 1) / 2 // half-block rows

	var lines []string
	for row := 0; row < rows; row++ {
		var sb strings.Builder
		// Left eye
		for col := 0; col < dspSize; col++ {
			topOn := isPixelOn(col*scale, row*2*scale, srcSize, mood, tick, 0)
			botOn := isPixelOn(col*scale, (row*2+1)*scale, srcSize, mood, tick, 0)
			sb.WriteRune(halfBlock(topOn, botOn))
		}
		// Gap
		for i := 0; i < gap; i++ {
			sb.WriteRune(' ')
		}
		// Right eye
		for col := 0; col < dspSize; col++ {
			topOn := isPixelOn(col*scale, row*2*scale, srcSize, mood, tick, 7777)
			botOn := isPixelOn(col*scale, (row*2+1)*scale, srcSize, mood, tick, 7777)
			sb.WriteRune(halfBlock(topOn, botOn))
		}
		lines = append(lines, sb.String())
	}

	// Color with accent
	var result []string
	for _, line := range lines {
		result = append(result, AccentStyle.Render(line))
	}
	return strings.Join(result, "\n")
}

func halfBlock(top, bot bool) rune {
	if top && bot {
		return '█'
	}
	if top {
		return '▀'
	}
	if bot {
		return '▄'
	}
	return ' '
}

func isPixelOn(x, y, size int, mood MascotMood, tick, seed int) bool {
	// Outside circle
	c := float64(size-1) / 2
	r := float64(size) / 2
	dx := float64(x) - c
	dy := float64(y) - c
	if dx*dx+dy*dy > r*r {
		return false
	}

	// Border
	isBorder := false
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := float64(x+d[0])-c, float64(y+d[1])-c
		if nx*nx+ny*ny > r*r {
			isBorder = true
			break
		}
	}
	if isBorder {
		return true
	}

	// Mood-specific eye parameters
	pupilRatio := 0.55
	topLid := 0.0     // how many pixels from top are erased (0 = fully open)
	bottomLid := 0.0   // how many pixels from bottom are erased

	switch mood {
	case MoodIdle:
		pupilRatio = 0.55
		topLid = 2 // slightly droopy
	case MoodThinking:
		pupilRatio = 0.65
		topLid = 8  // squinting from top
		bottomLid = 5 // and bottom
	case MoodExecuting:
		pupilRatio = 0.6
		topLid = 6
		bottomLid = 3
	case MoodDone:
		pupilRatio = 0.0 // no pupil, solid fill
		topLid = 10      // heavy lids = happy squint
		bottomLid = 4
	case MoodError:
		pupilRatio = 0.7 // wide pupils = startled
		topLid = 0       // eyes wide open
		bottomLid = 0
	}

	halfSize := float64(size) / 2.0

	// Top lid: erase pixels near the top of the circle
	if topLid > 0 {
		distFromTop := float64(y) - (c - r) // distance from top of circle
		if distFromTop < topLid {
			return false
		}
	}

	// Bottom lid: erase pixels near the bottom
	if bottomLid > 0 {
		distFromBottom := (c + r) - float64(y)
		if distFromBottom < bottomLid {
			return false
		}
	}

	_ = halfSize

	// Done mood: solid fill (no pupil, no noise -- just the eye shape)
	if mood == MoodDone {
		return true
	}

	// Error mood: X pattern inside the eye
	if mood == MoodError {
		cx, cy := float64(size)/2, float64(size)/2
		ex := float64(x) - cx
		ey := float64(y) - cy
		// X shape: pixels near the diagonals
		onDiag1 := math.Abs(ex-ey) < 2.5
		onDiag2 := math.Abs(ex+ey) < 2.5
		if onDiag1 || onDiag2 {
			return noise(x, y, tick*3+seed) // flickering X
		}
		return noise(x, y, tick+seed) && (tick+x+y)%3 == 0 // sparse noise
	}

	// Pupil hole
	if pupilRatio > 0 {
		pupilSize := float64(size) * pupilRatio
		pupilCenter := (float64(size) - pupilSize) / 2
		pdx := float64(x) - pupilCenter - pupilSize/2
		pdy := float64(y) - pupilCenter - pupilSize/2
		pr := pupilSize / 2
		if pdx*pdx+pdy*pdy <= pr*pr {
			return false
		}
	}

	// Noise fill
	return noise(x, y, tick+seed)
}

func noise(x, y, seed int) bool {
	h := uint32(x)*2654435761 + uint32(y)*2246822519 + uint32(seed)*3266489917
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	_ = math.Abs(0) // avoid unused import
	return (h % 100) < 50
}
