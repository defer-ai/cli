package tui

import (
	"math"
	"strings"
)

// MascotMood determines which expression to render.
type MascotMood int

const (
	MoodIdle   MascotMood = iota
	MoodActive            // thinking, executing, planning, verifying
	MoodAsking            // waiting for user input
	MoodDone
	MoodError
)

func StatusToMood(status string) MascotMood {
	switch status {
	case "thinking", "decomposing", "executing", "planning", "verifying":
		return MoodActive
	case "waiting":
		return MoodAsking
	case "done":
		return MoodDone
	case "error":
		return MoodError
	default:
		return MoodIdle
	}
}

// eyeFrame holds eye parameters for a single animation frame.
type eyeFrame struct {
	pupilRatio     float64
	pupilX         float64
	pupilY         float64
	topLid         float64
	topLidAngle    float64
	bottomLid      float64
	bottomLidAngle float64
	overlay        string  // "", "x", "twirl", "check"
	twirlAngle     float64
	checkOffsetX   float64
	checkOffsetY   float64
	sparkle        bool    // diamond sparkle in pupil
	solid          bool
	holdTicks      int // how many ticks to hold this frame (at 100ms/tick)
	transitionTicks int // ticks to interpolate to the next frame
}

// eyeAnim holds per-mood animation config.
type eyeAnim struct {
	lidRadius  float64
	cutoffMult float64
	frames     []eyeFrame
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func interpolateFrame(a, b eyeFrame, t float64) eyeFrame {
	return eyeFrame{
		pupilRatio:     lerp(a.pupilRatio, b.pupilRatio, t),
		pupilX:         lerp(a.pupilX, b.pupilX, t),
		pupilY:         lerp(a.pupilY, b.pupilY, t),
		topLid:         lerp(a.topLid, b.topLid, t),
		topLidAngle:    lerp(a.topLidAngle, b.topLidAngle, t),
		bottomLid:      lerp(a.bottomLid, b.bottomLid, t),
		bottomLidAngle: lerp(a.bottomLidAngle, b.bottomLidAngle, t),
		twirlAngle:     lerp(a.twirlAngle, b.twirlAngle, t),
		checkOffsetX:   lerp(a.checkOffsetX, b.checkOffsetX, t),
		checkOffsetY:   lerp(a.checkOffsetY, b.checkOffsetY, t),
		// Non-interpolatable: use source frame's values
		overlay: a.overlay,
		sparkle: a.sparkle,
		solid:   a.solid,
	}
}

// resolveFrame returns the interpolated frame for a given tick within a looping animation.
func resolveFrame(anim *eyeAnim, tick int) eyeFrame {
	if len(anim.frames) == 0 {
		return eyeFrame{}
	}
	if len(anim.frames) == 1 {
		f := anim.frames[0]
		if f.overlay == "twirl" {
			f.twirlAngle = float64(tick) * 0.17
		}
		return f
	}

	// Compute total ticks in one loop
	totalTicks := 0
	for _, f := range anim.frames {
		totalTicks += f.holdTicks + f.transitionTicks
	}
	if totalTicks == 0 {
		return anim.frames[0]
	}

	pos := tick % totalTicks
	accumulated := 0
	for i, f := range anim.frames {
		// Hold phase
		if pos < accumulated+f.holdTicks {
			return f
		}
		accumulated += f.holdTicks

		// Transition phase
		if f.transitionTicks > 0 && pos < accumulated+f.transitionTicks {
			next := anim.frames[(i+1)%len(anim.frames)]
			t := float64(pos-accumulated) / float64(f.transitionTicks)
			return interpolateFrame(f, next, t)
		}
		accumulated += f.transitionTicks
	}
	return anim.frames[0]
}

// --- Open/closed base frames ---
func openFrame() eyeFrame {
	return eyeFrame{
		pupilRatio: 0.65, topLid: 8, topLidAngle: 0,
		bottomLid: 0, bottomLidAngle: math.Pi,
	}
}

func blinkFrame() eyeFrame {
	return eyeFrame{
		pupilRatio: 0.65, topLid: 55, topLidAngle: 0,
		bottomLid: 20, bottomLidAngle: math.Pi,
	}
}

func withOverrides(base eyeFrame, overrides eyeFrame) eyeFrame {
	f := base
	if overrides.pupilRatio != 0 { f.pupilRatio = overrides.pupilRatio }
	if overrides.pupilX != 0 { f.pupilX = overrides.pupilX }
	if overrides.pupilY != 0 { f.pupilY = overrides.pupilY }
	if overrides.topLid != 0 { f.topLid = overrides.topLid }
	if overrides.topLidAngle != 0 { f.topLidAngle = overrides.topLidAngle }
	if overrides.bottomLid != 0 { f.bottomLid = overrides.bottomLid }
	if overrides.bottomLidAngle != 0 { f.bottomLidAngle = overrides.bottomLidAngle }
	if overrides.overlay != "" { f.overlay = overrides.overlay }
	if overrides.twirlAngle != 0 { f.twirlAngle = overrides.twirlAngle }
	if overrides.checkOffsetX != 0 { f.checkOffsetX = overrides.checkOffsetX }
	if overrides.checkOffsetY != 0 { f.checkOffsetY = overrides.checkOffsetY }
	f.solid = overrides.solid
	f.holdTicks = overrides.holdTicks
	f.transitionTicks = overrides.transitionTicks
	return f
}

// --- Animation definitions (ported from web eye-animation.ts) ---

var moodAnims = map[MascotMood]*eyeAnim{
	MoodIdle: {
		lidRadius: 3, cutoffMult: 1.2,
		frames: func() []eyeFrame {
			open := openFrame()
			blink := blinkFrame()
			return []eyeFrame{
				// Awake
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 8},
				// Blink
				{pupilRatio: 0.65, topLid: 55, bottomLid: 20, bottomLidAngle: math.Pi, holdTicks: 1, transitionTicks: 1},
				// Open
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 6, transitionTicks: 20},
				// Droop 1 — lids heavy, pupils drift up
				{pupilRatio: 0.65, topLid: 42, pupilY: -2, bottomLidAngle: math.Pi, holdTicks: 8, transitionTicks: 1},
				// Blink awake
				withOverrides(blink, eyeFrame{holdTicks: 1, transitionTicks: 1}),
				{pupilRatio: 0.65, topLid: 8, pupilY: -0.5, bottomLidAngle: math.Pi, holdTicks: 1, transitionTicks: 1},
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 4, transitionTicks: 40},
				// Droop 2 — heavier
				{pupilRatio: 0.65, topLid: 43, pupilY: -3, bottomLidAngle: math.Pi, holdTicks: 6, transitionTicks: 1},
				withOverrides(blink, eyeFrame{holdTicks: 1, transitionTicks: 1}),
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 3, transitionTicks: 50},
				// Droop 3 — pupils way up
				{pupilRatio: 0.65, topLid: 44, pupilX: -0.5, pupilY: -4, bottomLidAngle: math.Pi, holdTicks: 5, transitionTicks: 1},
				withOverrides(blink, eyeFrame{holdTicks: 1, transitionTicks: 1}),
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 2, transitionTicks: 60},
				// Give in — fully closed
				{pupilRatio: 0.65, topLid: 46, pupilY: -5, bottomLidAngle: math.Pi, holdTicks: 2, transitionTicks: 1},
				withOverrides(blink, eyeFrame{holdTicks: 15, transitionTicks: 1}),
				// SNAP awake — eyes wide, looking around
				withOverrides(open, eyeFrame{topLid: 0, pupilX: -3, pupilY: -1, holdTicks: 1, transitionTicks: 1}),
				withOverrides(open, eyeFrame{topLid: 0, pupilX: 3, pupilY: -1, holdTicks: 1, transitionTicks: 2}),
				withOverrides(open, eyeFrame{topLid: 0, pupilX: -2, pupilY: 1, holdTicks: 1, transitionTicks: 2}),
				withOverrides(open, eyeFrame{topLid: 0, pupilX: 2, holdTicks: 1, transitionTicks: 2}),
				withOverrides(open, eyeFrame{topLid: 0, holdTicks: 1, transitionTicks: 2}),
				// Calm down
				{pupilRatio: 0.65, topLid: 8, bottomLidAngle: math.Pi, holdTicks: 4, transitionTicks: 6},
			}
		}(),
	},

	MoodActive: {
		lidRadius: 2.5, cutoffMult: 1.2,
		frames: func() []eyeFrame {
			// Rotating twirl — 12 steps of rotation
			var frames []eyeFrame
			steps := 12
			for i := 0; i < steps; i++ {
				angle := float64(i) / float64(steps) * math.Pi * 2
				frames = append(frames, eyeFrame{
					pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5,
					overlay: "twirl", twirlAngle: angle,
					holdTicks: 1, transitionTicks: 0,
				})
			}
			// Glitchy jitter — pupil changes only, lids stay locked at 37.5
			frames = append(frames,
				eyeFrame{pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5, pupilX: 2, overlay: "twirl", twirlAngle: 1.8, holdTicks: 1},
				eyeFrame{pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5, pupilX: -1.5, overlay: "twirl", twirlAngle: 2.0, holdTicks: 1},
				eyeFrame{pupilRatio: 0.4, topLid: 37.5, bottomLid: 37.5, pupilX: 0.5, pupilY: -1, overlay: "twirl", twirlAngle: 4.0, holdTicks: 1},
				eyeFrame{pupilRatio: 0.9, topLid: 37.5, bottomLid: 37.5, pupilX: -0.5, pupilY: 0.5, overlay: "twirl", twirlAngle: 5.5, holdTicks: 1},
				eyeFrame{pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5, overlay: "twirl", twirlAngle: 0, holdTicks: 2, transitionTicks: 1},
			)
			return frames
		}(),
	},

	MoodDone: {
		lidRadius: 4, cutoffMult: 1.2,
		frames: func() []eyeFrame {
			var frames []eyeFrame
			// Check facing one way, oscillating offset
			offsets := [][2]float64{{-4.5, 1.0}, {-4.4, 1.1}, {-4.3, 1.2}, {-4.2, 1.3}, {-4.1, 1.4}, {-4.0, 1.5}}
			for _, o := range offsets {
				frames = append(frames, eyeFrame{
					pupilRatio: 0, overlay: "check", solid: true,
					topLid: 60, topLidAngle: -0.4, bottomLidAngle: math.Pi,
					checkOffsetX: o[0], checkOffsetY: o[1],
					holdTicks: 2,
				})
			}
			// Blink
			frames = append(frames, eyeFrame{
				pupilRatio: 0, overlay: "check", solid: true,
				topLid: 85, topLidAngle: -0.4, bottomLid: 15, bottomLidAngle: math.Pi,
				checkOffsetX: -4.0, checkOffsetY: 1.5,
				holdTicks: 1, transitionTicks: 2,
			})
			// Open, oscillate back
			for i := len(offsets) - 1; i >= 0; i-- {
				o := offsets[i]
				frames = append(frames, eyeFrame{
					pupilRatio: 0, overlay: "check", solid: true,
					topLid: 60, topLidAngle: -0.4, bottomLidAngle: math.Pi,
					checkOffsetX: o[0], checkOffsetY: o[1],
					holdTicks: 2,
				})
			}
			// Blink
			frames = append(frames, eyeFrame{
				pupilRatio: 0, overlay: "check", solid: true,
				topLid: 85, topLidAngle: -0.4, bottomLid: 15, bottomLidAngle: math.Pi,
				checkOffsetX: -4.5, checkOffsetY: 1.0,
				holdTicks: 1, transitionTicks: 2,
			})
			return frames
		}(),
	},

	MoodAsking: {
		lidRadius: 2.5, cutoffMult: 1.2,
		frames: []eyeFrame{
			// Wide-eyed curious look — big pupils, top lid angled up, bottom lid raised, diamond sparkle
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			// Double blink
			{pupilRatio: 0.85, topLid: 18, topLidAngle: math.Pi, bottomLid: 52, holdTicks: 1, transitionTicks: 1},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 3},
			{pupilRatio: 0.85, topLid: 18, topLidAngle: math.Pi, bottomLid: 52, holdTicks: 1, transitionTicks: 1},
			{pupilRatio: 0.85, topLid: 0, topLidAngle: math.Pi, bottomLid: 38, sparkle: true, holdTicks: 5},
		},
	},

	MoodError: {
		lidRadius: 2.5, cutoffMult: 1.2,
		frames: []eyeFrame{
			// X with wandering pupil (no blink)
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, holdTicks: 5, transitionTicks: 3},
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, pupilX: -2, pupilY: -1, holdTicks: 5, transitionTicks: 4},
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, pupilX: 2, pupilY: 0.5, holdTicks: 5, transitionTicks: 3},
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, pupilX: 1, pupilY: -1.5, holdTicks: 5, transitionTicks: 3},
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, pupilX: -1.5, pupilY: 1, holdTicks: 5, transitionTicks: 4},
			{pupilRatio: 0.75, overlay: "x", topLid: 4, bottomLidAngle: math.Pi, pupilX: 0.5, pupilY: -0.5, holdTicks: 5, transitionTicks: 3},
		},
	},
}

const (
	srcSize     = 30
	displaySize = 15
	eyeGap      = 4
)

// RenderMascot returns a multi-line string of the mascot at the given tick.
func RenderMascot(mood MascotMood, tick int) string {
	return RenderMascotAtSize(mood, tick, displaySize, eyeGap)
}

// RenderMascotAtSize renders the mascot at a custom display size.
func RenderMascotAtSize(mood MascotMood, tick, dspSize, gap int) string {
	anim := moodAnims[mood]
	frame := resolveFrame(anim, tick)

	rows := (dspSize + 1) / 2

	var lines []string
	for row := 0; row < rows; row++ {
		var sb strings.Builder
		for col := 0; col < dspSize; col++ {
			sx := col * srcSize / dspSize
			topY := (row * 2) * srcSize / dspSize
			botY := (row*2 + 1) * srcSize / dspSize
			topOn := renderPixel(sx, topY, srcSize, frame, anim, tick, 0, false)
			botOn := renderPixel(sx, botY, srcSize, frame, anim, tick, 0, false)
			sb.WriteRune(halfBlock(topOn, botOn))
		}
		for i := 0; i < gap; i++ {
			sb.WriteRune(' ')
		}
		mirrorFrame := frame
		mirrorFrame.topLidAngle = -frame.topLidAngle
		for col := 0; col < dspSize; col++ {
			sx := col * srcSize / dspSize
			topY := (row * 2) * srcSize / dspSize
			botY := (row*2 + 1) * srcSize / dspSize
			topOn := renderPixel(sx, topY, srcSize, mirrorFrame, anim, tick, 7777, true)
			botOn := renderPixel(sx, botY, srcSize, mirrorFrame, anim, tick, 7777, true)
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

func renderPixel(x, y, size int, frame eyeFrame, anim *eyeAnim, tick, seed int, mirror bool) bool {
	c := float64(size-1) / 2.0
	r := float64(size) / 2.0
	lidR := r * anim.lidRadius
	fx, fy := float64(x), float64(y)

	if !isInCircle(fx, fy, c, r) {
		return false
	}

	topLid := computeLid(fx, fy, c, r, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true)
	botLid := computeLid(fx, fy, c, r, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false)

	inTopCrescent := topLid.inLid && !topLid.inCut
	inBotCrescent := botLid.inLid && !botLid.inCut

	if inTopCrescent || inBotCrescent {
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := float64(x+d[0]), float64(y+d[1])
			if !isInCircle(nx, ny, c, r) {
				continue
			}
			nTop := computeLid(nx, ny, c, r, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true)
			nBot := computeLid(nx, ny, c, r, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false)
			if !(nTop.inLid && !nTop.inCut) && !(nBot.inLid && !nBot.inCut) {
				return true
			}
		}
		return false
	}

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

	if frame.pupilRatio > 0 {
		pupilSize := float64(size) * frame.pupilRatio
		pupilCenter := (float64(size) - pupilSize) / 2.0
		px := fx - pupilCenter - frame.pupilX
		py := fy - pupilCenter - frame.pupilY
		pr := pupilSize / 2.0
		inPupil := (px-pr)*(px-pr)+(py-pr)*(py-pr) <= pr*pr

		if inPupil {
			// Diamond sparkle in pupil (asking mood)
			// Rendered in source-space (30px). The caller downsamples, so:
			// - At 15/20px display: only the center pixel survives → 1px glimmer
			// - At 30px display: full 5px cross visible
			if frame.sparkle {
				sc := float64(size-1) / 2.0
				spx := math.Floor(sc + frame.pupilX - pupilSize*0.25)
				spy := math.Floor(sc + frame.pupilY - pupilSize*0.25)
				sdx := int(math.Round(fx - spx))
				sdy := int(math.Round(fy - spy))
				phase := (tick / 3) % 4
				switch phase {
				case 0:
					if sdx == 0 && sdy == 0 {
						return true
					}
				case 1:
					// 5px cross at source resolution
					if (sdx == 0 && sdy == 0) ||
						(sdx == 0 && sdy == -1) ||
						(sdx == 0 && sdy == 1) ||
						(sdx == -1 && sdy == 0) ||
						(sdx == 1 && sdy == 0) {
						return true
					}
				case 2:
					if sdx == 0 && sdy == 0 {
						return true
					}
				}
			}

			if frame.overlay == "x" {
				gapInner := pupilSize - 2
				distFromCenter := math.Hypot(
					fx-(pupilCenter+frame.pupilX+pupilSize/2),
					fy-(pupilCenter+frame.pupilY+pupilSize/2),
				)
				if distFromCenter > gapInner/2 {
					return false
				}
				if isXPixel(fx-frame.pupilX, fy-frame.pupilY, float64(size)) {
					return noiseHash(x, y, tick*3+seed, 75)
				}
				return false
			}
			if frame.overlay == "twirl" {
				if isTwirlPixel(fx, fy, float64(size), frame.twirlAngle) {
					return noiseHash(x, y, tick/8+seed, 80)
				}
			}
			// Pupil hole: transparent, but not during blinks
			// (prevents empty gap in the middle of closed eyes)
			if (frame.topLid + frame.bottomLid) <= 50 {
				return false
			}
		}
	}

	if frame.overlay == "check" {
		if isCheckPixelOffset(fx, fy, float64(size), mirror, frame.checkOffsetX, frame.checkOffsetY) {
			return false
		}
	}

	if frame.solid {
		return true
	}
	if frame.overlay == "x" {
		return noiseHash(x, y, tick+seed, 10)
	}
	if frame.overlay == "twirl" {
		return noiseHash(x, y, tick+seed, 20)
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

func isCheckPixelOffset(x, y, size float64, mirror bool, offsetX, offsetY float64) bool {
	s := size * 0.3
	dentX := size/2 + offsetX
	dentY := size*0.75 + offsetY
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
	return math.Min(d1, d2) < 3.5 // thicker stroke for visibility at small sizes
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
