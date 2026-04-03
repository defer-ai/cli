package tui

import (
	"math"
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

// eyeFrame holds per-mood eye parameters (ported from web eye-animation.ts)
type eyeFrame struct {
	pupilRatio    float64
	pupilX        float64
	pupilY        float64
	topLid        float64
	topLidAngle   float64
	bottomLid     float64
	bottomLidAngle float64
	overlay       string // "", "x", "twirl", "check"
	twirlAngle    float64
	solid         bool
}

// eyeAnim holds per-mood animation config
type eyeAnim struct {
	lidRadius float64
	cutoffMult float64
}

// Static frames per mood (representative frame from each expression)
var moodFrames = map[MascotMood]eyeFrame{
	MoodIdle: {
		pupilRatio: 0.65, topLid: 8, topLidAngle: 0,
		bottomLid: 0, bottomLidAngle: math.Pi,
	},
	MoodActive: {
		pupilRatio: 0.75, topLid: 37.5, topLidAngle: 0,
		bottomLid: 37.5, bottomLidAngle: 0,
		overlay: "twirl",
	},
	MoodDone: {
		pupilRatio: 0.4, topLid: 40, topLidAngle: 0,
		bottomLid: 35, bottomLidAngle: 0,
	},
	MoodError: {
		pupilRatio: 0.75, topLid: 4, topLidAngle: 0,
		bottomLid: 0, bottomLidAngle: math.Pi,
		overlay: "x",
	},
}

var moodAnims = map[MascotMood]eyeAnim{
	MoodIdle:   {lidRadius: 3, cutoffMult: 1.2},
	MoodActive: {lidRadius: 2.5, cutoffMult: 1.2},
	MoodDone:   {lidRadius: 4, cutoffMult: 1.2},
	MoodError:  {lidRadius: 2.5, cutoffMult: 1.2},
}

const (
	srcSize = 28
	scale   = 4
	dspSize = srcSize / scale
	eyeGap  = 2
)

// RenderMascot returns a multi-line string of the mascot at the given tick.
func RenderMascot(mood MascotMood, tick int) string {
	frame := moodFrames[mood]
	anim := moodAnims[mood]

	// Animate twirl angle for active mood
	if mood == MoodActive {
		frame.twirlAngle = float64(tick) * 0.17 // ~10 degrees per tick
	}

	rows := (dspSize + 1) / 2

	var lines []string
	for row := 0; row < rows; row++ {
		var sb strings.Builder
		// Left eye
		for col := 0; col < dspSize; col++ {
			topOn := renderPixel(col*scale, row*2*scale, srcSize, frame, anim, tick, 0, false)
			botOn := renderPixel(col*scale, (row*2+1)*scale, srcSize, frame, anim, tick, 0, false)
			sb.WriteRune(halfBlock(topOn, botOn))
		}
		// Gap
		for i := 0; i < eyeGap; i++ {
			sb.WriteRune(' ')
		}
		// Right eye (mirrored topLidAngle)
		mirrorFrame := frame
		mirrorFrame.topLidAngle = -frame.topLidAngle
		for col := 0; col < dspSize; col++ {
			topOn := renderPixel(col*scale, row*2*scale, srcSize, mirrorFrame, anim, tick, 7777, true)
			botOn := renderPixel(col*scale, (row*2+1)*scale, srcSize, mirrorFrame, anim, tick, 7777, true)
			sb.WriteRune(halfBlock(topOn, botOn))
		}
		lines = append(lines, sb.String())
	}

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

// --- Core pixel renderer (ported from web mascot.tsx) ---

func renderPixel(x, y, size int, frame eyeFrame, anim eyeAnim, tick, seed int, mirror bool) bool {
	c := float64(size-1) / 2.0
	r := float64(size) / 2.0
	lidR := r * anim.lidRadius
	fx, fy := float64(x), float64(y)

	// Outside circle
	if !isInCircle(fx, fy, c, r) {
		return false
	}

	// Compute lids
	topLid := computeLid(fx, fy, c, r, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true)
	botLid := computeLid(fx, fy, c, r, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false)

	inTopCrescent := topLid.inLid && !topLid.inCut
	inBotCrescent := botLid.inLid && !botLid.inCut

	// Lid crescent: show border or transparent
	if inTopCrescent || inBotCrescent {
		// Check if this is a lid border pixel
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := float64(x+d[0]), float64(y+d[1])
			if !isInCircle(nx, ny, c, r) {
				continue
			}
			nTop := computeLid(nx, ny, c, r, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true)
			nBot := computeLid(nx, ny, c, r, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false)
			if !(nTop.inLid && !nTop.inCut) && !(nBot.inLid && !nBot.inCut) {
				return true // lid border
			}
		}
		return false // inside lid, not border
	}

	// Eye border
	if isCircleBorder(fx, fy, c, r) {
		if frame.topLid > 0 || frame.bottomLid > 0 {
			tc := computeLid(fx, fy, c, r, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true)
			bc := computeLid(fx, fy, c, r, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false)
			if (tc.inLid && !tc.inCut) || (bc.inLid && !bc.inCut) {
				return false
			}
		}
		return true
	}

	// Pupil
	if frame.pupilRatio > 0 {
		pupilSize := float64(size) * frame.pupilRatio
		pupilCenter := (float64(size) - pupilSize) / 2.0
		px := fx - pupilCenter - frame.pupilX
		py := fy - pupilCenter - frame.pupilY
		pr := pupilSize / 2.0
		inPupil := (px-pr)*(px-pr)+(py-pr)*(py-pr) <= pr*pr

		if inPupil {
			// X inside pupil
			if frame.overlay == "x" {
				gapInner := pupilSize - 3
				distFromCenter := math.Hypot(
					fx-(pupilCenter+frame.pupilX+pupilSize/2),
					fy-(pupilCenter+frame.pupilY+pupilSize/2),
				)
				if distFromCenter > gapInner/2 {
					return false // gap ring
				}
				if isXPixel(fx-frame.pupilX, fy-frame.pupilY, float64(size)) {
					return noiseHash(x, y, tick*5+seed, 60)
				}
				return false
			}
			// Twirl inside pupil
			if frame.overlay == "twirl" {
				if isTwirlPixel(fx, fy, float64(size), frame.twirlAngle) {
					return noiseHash(x, y, tick/8+seed, 80)
				}
			}
			return false // transparent pupil
		}
	}

	// Check overlay outside pupil
	if frame.overlay == "check" {
		if isCheckPixel(fx, fy, float64(size), mirror) {
			return false
		}
	}

	// Fill
	if frame.solid {
		return true
	}
	if frame.overlay == "x" || frame.overlay == "twirl" {
		return noiseHash(x, y, tick+seed, 20) // sparse 20%
	}
	return noise(x, y, tick+seed)
}

// --- Math helpers ---

func isInCircle(x, y, c, r float64) bool {
	dx, dy := x-c, y-c
	return dx*dx+dy*dy <= r*r
}

func isCircleBorder(x, y, c, r float64) bool {
	if !isInCircle(x, y, c, r) {
		return false
	}
	return !isInCircle(x-1, y, c, r) || !isInCircle(x+1, y, c, r) ||
		!isInCircle(x, y-1, c, r) || !isInCircle(x, y+1, c, r)
}

type lidResult struct {
	inLid bool
	inCut bool
}

func computeLid(x, y, c, r, travel, angle, lidR, cutoffMult float64, fromTop bool) lidResult {
	startOffset := lidR + r
	cutDist := lidR * cutoffMult

	var lidCy, cutX, cutY float64
	if fromTop {
		lidCy = c - startOffset + travel
		cutX = c + math.Sin(angle)*cutDist
		cutY = lidCy + math.Cos(angle)*cutDist
	} else {
		lidCy = c + startOffset - travel
		cutX = c - math.Sin(angle)*cutDist
		cutY = lidCy - math.Cos(angle)*cutDist
	}

	dx1, dy1 := x-c, y-lidCy
	inLid := dx1*dx1+dy1*dy1 <= lidR*lidR

	dx2, dy2 := x-cutX, y-cutY
	inCut := dx2*dx2+dy2*dy2 <= lidR*lidR

	return lidResult{inLid: inLid, inCut: inCut}
}

func distToSeg(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		return math.Hypot(px-x1, py-y1)
	}
	t := math.Max(0, math.Min(1, ((px-x1)*dx+(py-y1)*dy)/lenSq))
	return math.Hypot(px-(x1+t*dx), py-(y1+t*dy))
}

func isCheckPixel(x, y, size float64, mirror bool) bool {
	s := size * 0.3
	dentX := size / 2
	dentY := size * 0.75
	dir := 1.0
	d1 := distToSeg(x, y, dentX-dir*s*0.35, dentY-s*0.3, dentX, dentY)
	d2 := distToSeg(x, y, dentX, dentY, dentX+dir*s*0.55, dentY-s*0.9)
	return math.Min(d1, d2) < 1.5
}

func isXPixel(x, y, size float64) bool {
	s := size * 0.42
	cx, cy := size/2, size/2
	d1 := distToSeg(x, y, cx-s, cy-s, cx+s, cy+s)
	d2 := distToSeg(x, y, cx+s, cy-s, cx-s, cy+s)
	return math.Min(d1, d2) < 2.5
}

func isTwirlPixel(x, y, size, angle float64) bool {
	cx, cy := size/2, size/2
	dx, dy := x-cx, y-cy
	dist := math.Hypot(dx, dy)
	if dist < 0.5 {
		return false
	}
	pixelAngle := math.Atan2(dy, dx)
	spiral := pixelAngle + math.Log(dist+1)*2.5 + angle
	stripe := math.Mod(spiral/math.Pi, 1)
	if stripe < 0 {
		stripe += 1
	}
	return stripe < 0.5
}

func noise(x, y, seed int) bool {
	return noiseHash(x, y, seed, 50)
}

func noiseHash(x, y, seed, threshold int) bool {
	h := uint32(x)*2654435761 + uint32(y)*2246822519 + uint32(seed)*3266489917
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return (h % 100) < uint32(threshold)
}
