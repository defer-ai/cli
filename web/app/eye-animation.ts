// Eye animation system
// Each expression is a sequence of frames with explicit parameters per frame.
// The renderer interpolates nothing - it shows exactly what each frame says.

export interface EasingCurve {
  attack: number;   // 0-1, how fast it accelerates at the start
  decay: number;    // 0-1, how fast it decelerates at the end
}

export interface EyeFrame {
  // Pupil
  pupilRatio: number;       // 0-1, size of transparent pupil hole
  pupilX: number;           // horizontal offset of pupil (-5 to 5)
  pupilY: number;           // vertical offset of pupil (-5 to 5)

  // Top eyelid
  topLid: number;           // how far top lid has traveled in (0 = open)
  topLidAngle: number;      // rotation of top lid cutout (radians)

  // Bottom eyelid
  bottomLid: number;        // how far bottom lid has traveled in (0 = open)
  bottomLidAngle: number;   // rotation of bottom lid cutout (radians)

  // Overlay (transparent shape inside noise)
  overlay?: "check" | "check-flip" | "x" | "twirl";

  // Twirl rotation angle (radians), only used when overlay is "twirl"
  twirlAngle?: number;

  // Sparkle type in the pupil
  sparkle?: "blink" | "wink" | "diamond" | "sweep" | "cluster";

  // Move checkmark position
  checkOffsetX?: number;
  checkOffsetY?: number;

  // Fill
  solid?: boolean;          // solid color instead of noise

  // How long to hold this frame before transitioning (overrides global frameDuration)
  hold?: number;

  // Transition to the next frame
  transition?: {
    duration: number;       // ms to transition to next frame
    easing: EasingCurve;    // curve shape
  };
}

export interface EyeAnimation {
  name: string;
  eyeSize: number;
  gap: number;              // gap between eyes in pixels
  lidRadius: number;        // lid circle size multiplier
  cutoffMult: number;       // crescent thickness control
  noiseSpeed: number;       // ms per noise tick
  frameDuration: number;    // ms per animation frame
  loop: boolean;            // loop the animation
  frames: EyeFrame[];
}

// --- Easing ---

/**
 * Custom easing curve using attack/decay.
 * attack controls how fast it leaves the start (high = snappy start)
 * decay controls how fast it arrives at the end (high = snappy end)
 * t is 0-1 progress
 */
export function applyEasing(t: number, curve: EasingCurve): number {
  const { attack, decay } = curve;
  // Use a combination of power curves
  // attack controls the ease-out at start, decay controls the ease-in at end
  const a = 1 + attack * 3;  // 1 to 4
  const d = 1 + decay * 3;   // 1 to 4
  // Generalized sigmoid-like: fast start (high attack) + fast end (high decay)
  const startCurve = Math.pow(t, 1 / a);
  const endCurve = 1 - Math.pow(1 - t, 1 / d);
  // Blend both curves
  return (startCurve + endCurve) / 2;
}

/**
 * Interpolate between two frames at progress t (0-1)
 */
export function interpolateFrames(a: EyeFrame, b: EyeFrame, t: number): EyeFrame {
  const lerp = (from: number, to: number) => from + (to - from) * t;
  return {
    pupilRatio: lerp(a.pupilRatio, b.pupilRatio),
    pupilX: lerp(a.pupilX, b.pupilX),
    pupilY: lerp(a.pupilY, b.pupilY),
    topLid: lerp(a.topLid, b.topLid),
    topLidAngle: lerp(a.topLidAngle, b.topLidAngle),
    bottomLid: lerp(a.bottomLid, b.bottomLid),
    bottomLidAngle: lerp(a.bottomLidAngle, b.bottomLidAngle),
    // Non-interpolatable props: use target frame's values
    overlay: t > 0.5 ? b.overlay : a.overlay,
    solid: t > 0.5 ? b.solid : a.solid,
    sparkle: t > 0.5 ? b.sparkle : a.sparkle,
    twirlAngle: lerp(a.twirlAngle ?? 0, b.twirlAngle ?? 0),
    checkOffsetX: lerp(a.checkOffsetX ?? 0, b.checkOffsetX ?? 0),
    checkOffsetY: lerp(a.checkOffsetY ?? 0, b.checkOffsetY ?? 0),
    transition: b.transition,
  };
}

// Common easing presets
export const EASE = {
  linear: { attack: 0, decay: 0 } as EasingCurve,
  smooth: { attack: 0.5, decay: 0.5 } as EasingCurve,
  snappy: { attack: 0.9, decay: 0.9 } as EasingCurve,
  sluggish: { attack: 0.2, decay: 0.2 } as EasingCurve,
  snapIn: { attack: 0.1, decay: 0.9 } as EasingCurve,   // slow start, fast end
  snapOut: { attack: 0.9, decay: 0.1 } as EasingCurve,   // fast start, slow end
  sleepy: { attack: 0.1, decay: 0.3 } as EasingCurve,
};

// --- Expression definitions ---

const OPEN_FRAME: EyeFrame = {
  pupilRatio: 0.65,
  pupilX: 0,
  pupilY: 0,
  topLid: 4,
  topLidAngle: 0,
  bottomLid: 0,
  bottomLidAngle: Math.PI,
};

const CLOSED_FRAME: EyeFrame = {
  pupilRatio: 0.65,
  pupilX: 0,
  pupilY: 0,
  topLid: 50,
  topLidAngle: 0,
  bottomLid: 50,
  bottomLidAngle: 0,
};

function frame(overrides: Partial<EyeFrame>): EyeFrame {
  return { ...OPEN_FRAME, ...overrides };
}

export const EXPRESSIONS: Record<string, EyeAnimation> = {
  idle: {
    name: "idle",
    eyeSize: 30,
    gap: 8,
    lidRadius: 3,
    cutoffMult: 1.2,
    noiseSpeed: 100,
    frameDuration: 400,
    loop: true,
    frames: [
      // Awake, normal
      frame({ topLid: 8, bottomLid: 0, hold: 800 }),

      // Normal blink
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 8, bottomLid: 0, hold: 600, transition: { duration: 2000, easing: EASE.sleepy } }),

      // Droop 1, pupils drift up, blink awake
      frame({ topLid: 42, bottomLid: 0, pupilY: -2, hold: 800, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: -0.5, hold: 100, transition: { duration: 120, easing: EASE.smooth } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: 0, hold: 400, transition: { duration: 4000, easing: EASE.sleepy } }),

      // Droop 2, pupils higher, blink awake
      frame({ topLid: 43, bottomLid: 0, pupilY: -3, hold: 600, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: -0.8, hold: 100, transition: { duration: 120, easing: EASE.smooth } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: 0, hold: 300, transition: { duration: 5000, easing: EASE.sleepy } }),

      // Droop 3, pupils way up, blink awake
      frame({ topLid: 44, bottomLid: 0, pupilX: -0.5, pupilY: -4, hold: 500, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: -1, hold: 100, transition: { duration: 120, easing: EASE.smooth } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: 0, hold: 200, transition: { duration: 6000, easing: EASE.sleepy } }),

      // Droop 4, blink awake
      frame({ topLid: 45, bottomLid: 0, pupilY: -4.5, hold: 400, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: -1.2, hold: 100, transition: { duration: 120, easing: EASE.smooth } }),
      frame({ topLid: 8, bottomLid: 0, pupilY: 0, hold: 200, transition: { duration: 7000, easing: EASE.sleepy } }),

      // Pause before giving in
      frame({ topLid: 8, bottomLid: 0, hold: 500, transition: { duration: 8000, easing: EASE.sleepy } }),

      // Can't fight it, pupils rolling up
      frame({ topLid: 46, bottomLid: 0, pupilY: -5 }),

      // Fully closed, hold
      frame({ topLid: 49.5, bottomLid: 15.5, bottomLidAngle: Math.PI, hold: 1500, transition: { duration: 120, easing: EASE.snapOut } }),

      // SNAP awake - eyes wide, looking around scared
      frame({ topLid: 0, bottomLid: 0, pupilX: -3, pupilY: -1, hold: 100, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ topLid: 0, bottomLid: 0, pupilX: 3, pupilY: -1, transition: { duration: 150, easing: EASE.snappy } }),
      frame({ topLid: 0, bottomLid: 0, pupilX: -2, pupilY: 1, transition: { duration: 150, easing: EASE.snappy } }),
      frame({ topLid: 0, bottomLid: 0, pupilX: 2, pupilY: 0, transition: { duration: 150, easing: EASE.snappy } }),
      frame({ topLid: 0, bottomLid: 0, pupilX: 0, pupilY: 0, transition: { duration: 200, easing: EASE.smooth } }),

      // Calming down
      frame({ topLid: 8, bottomLid: 0, pupilY: 0, transition: { duration: 600, easing: EASE.smooth } }),
    ],
  },

  thinking: {
    name: "thinking",
    eyeSize: 30,
    gap: 8,
    lidRadius: 2.5,
    cutoffMult: 1.2,
    noiseSpeed: 40,
    frameDuration: 50,
    loop: true,
    frames: (() => {
      // Rotating twirl overlay
      const steps = 36;
      const f: EyeFrame[] = [];
      for (let i = 0; i < steps; i++) {
        const angle = (i / steps) * Math.PI * 2;
        f.push(frame({
          pupilRatio: 0.75,
          topLid: 37.5,
          topLidAngle: 0,
          bottomLid: 37.5,
          bottomLidAngle: 0,
          overlay: "twirl",
          twirlAngle: angle,
        }));
      }
      // Glitchy "processing" blink: stuttery jitter, lids stay near baseline
      f[f.length - 1] = { ...f[f.length - 1], transition: { duration: 40, easing: EASE.snappy } };

      // Pupil jumps right, slight lid twitch
      f.push(frame({ pupilRatio: 0.75, topLid: 40, bottomLid: 37.5, pupilX: 2, overlay: "twirl", twirlAngle: 1.8, hold: 30, transition: { duration: 30, easing: EASE.snappy } }));
      // Twirl frozen at wrong angle, pupil snaps left
      f.push(frame({ pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5, pupilX: -1.5, overlay: "twirl", twirlAngle: 2.0, hold: 50, transition: { duration: 30, easing: EASE.snappy } }));
      // Quick shut-open flash (only frame that closes hard)
      f.push(frame({ pupilRatio: 0.75, topLid: 50, bottomLid: 45, hold: 15, transition: { duration: 25, easing: EASE.snappy } }));
      // Snap open, pupil tiny + drifts, twirl jumps
      f.push(frame({ pupilRatio: 0.4, topLid: 38, bottomLid: 37.5, pupilX: 0.5, pupilY: -1, overlay: "twirl", twirlAngle: 4.0, hold: 40, transition: { duration: 30, easing: EASE.snappy } }));
      // Pupil overshoots big
      f.push(frame({ pupilRatio: 0.9, topLid: 37.5, bottomLid: 37.5, pupilX: -0.5, pupilY: 0.5, overlay: "twirl", twirlAngle: 5.5, hold: 35, transition: { duration: 40, easing: EASE.snappy } }));
      // Tiny stutter
      f.push(frame({ pupilRatio: 0.75, topLid: 39, bottomLid: 38, pupilX: 1, hold: 20, transition: { duration: 25, easing: EASE.snappy } }));
      // Recover to normal
      f.push(frame({ pupilRatio: 0.75, topLid: 37.5, bottomLid: 37.5, overlay: "twirl", twirlAngle: 0, transition: { duration: 80, easing: EASE.smooth } }));
      return f;
    })(),
  },

  asking: {
    name: "asking",
    eyeSize: 30,
    gap: 8,
    lidRadius: 2.5,
    cutoffMult: 1.2,
    noiseSpeed: 100,
    frameDuration: 300,
    loop: true,
    frames: [
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      // Blink 1
      frame({ pupilRatio: 0.85, topLid: 13, topLidAngle: Math.PI, bottomLid: 46, bottomLidAngle: 0, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
      // Blink 2
      frame({ pupilRatio: 0.85, topLid: 13, topLidAngle: Math.PI, bottomLid: 46, bottomLidAngle: 0, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ pupilRatio: 0.85, topLid: 0, topLidAngle: Math.PI, bottomLid: 38, bottomLidAngle: 0, sparkle: "diamond" }),
    ],
  },

  done: {
    name: "done",
    eyeSize: 30,
    gap: 8,
    lidRadius: 4,
    cutoffMult: 1.2,
    noiseSpeed: 150,
    frameDuration: 300,
    loop: true,
    frames: [
      // Check facing one way, cycling checkOffsetX -4.5 -> -4.0, checkOffsetY 1 -> 1.5
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.5, checkOffsetY: 1.0 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.4, checkOffsetY: 1.1 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.3, checkOffsetY: 1.2 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.2, checkOffsetY: 1.3 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.1, checkOffsetY: 1.4 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.0, checkOffsetY: 1.5, transition: { duration: 120, easing: EASE.snappy } }),
      // Blink (closed)
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 80, topLidAngle: -0.4, bottomLid: 9, bottomLidAngle: Math.PI, checkOffsetX: -4.0, checkOffsetY: 1.5, hold: 50, transition: { duration: 200, easing: EASE.smooth } }),
      // After blink, cycling back
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.0, checkOffsetY: 1.5, transition: { duration: 200, easing: EASE.smooth } }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.1, checkOffsetY: 1.4 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.2, checkOffsetY: 1.3 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.3, checkOffsetY: 1.2 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.4, checkOffsetY: 1.1 }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.5, checkOffsetY: 1.0, transition: { duration: 120, easing: EASE.snappy } }),
      // Blink (closed)
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 80, topLidAngle: -0.4, bottomLid: 9, bottomLidAngle: Math.PI, checkOffsetX: -4.5, checkOffsetY: 1.0, hold: 50, transition: { duration: 200, easing: EASE.smooth } }),
      frame({ pupilRatio: 0, overlay: "check", solid: true, topLid: 60, topLidAngle: -0.4, checkOffsetX: -4.5, checkOffsetY: 1.0, transition: { duration: 200, easing: EASE.smooth } }),
    ],
  },

  error: {
    name: "error",
    eyeSize: 30,
    gap: 8,
    lidRadius: 2.5,
    cutoffMult: 1.2,
    noiseSpeed: 40,
    frameDuration: 500,
    loop: true,
    frames: [
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: 0, pupilY: 0, transition: { duration: 300, easing: EASE.smooth } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: -2, pupilY: -1, transition: { duration: 400, easing: EASE.smooth } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: 2, pupilY: 0.5, transition: { duration: 350, easing: EASE.smooth } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: 1, pupilY: -1.5, transition: { duration: 300, easing: EASE.smooth } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: -1.5, pupilY: 1, transition: { duration: 400, easing: EASE.smooth } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: 0.5, pupilY: -0.5, transition: { duration: 80, easing: EASE.snappy } }),
      // Blink
      frame({ pupilRatio: 0.75, overlay: "x", topLid: 60, bottomLid: 60, hold: 10, transition: { duration: 80, easing: EASE.snappy } }),
      frame({ pupilRatio: 0.75, overlay: "x", pupilX: 0, pupilY: 0, transition: { duration: 80, easing: EASE.snappy } }),
    ],
  },
};
