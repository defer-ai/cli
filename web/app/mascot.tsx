"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { EXPRESSIONS, type EyeAnimation, type EyeFrame, applyEasing, interpolateFrames } from "./eye-animation";

// --- Color ---
const ACCENT = "#f97316";

// --- Noise ---
function noise(x: number, y: number, seed: number): boolean {
  let h = (x * 2654435761 + y * 2246822519 + seed * 3266489917) >>> 0;
  h ^= h >> 16;
  h = (h * 2246822507) >>> 0;
  h ^= h >> 13;
  h = (h * 3266489909) >>> 0;
  h ^= h >> 16;
  return (h % 100) < 50;
}

// --- Circle shape ---
function isInCircle(x: number, y: number, size: number): boolean {
  const c = (size - 1) / 2;
  const r = size / 2;
  return (x - c) * (x - c) + (y - c) * (y - c) <= r * r;
}

function isCircleBorder(x: number, y: number, size: number): boolean {
  if (!isInCircle(x, y, size)) return false;
  return (
    !isInCircle(x - 1, y, size) ||
    !isInCircle(x + 1, y, size) ||
    !isInCircle(x, y - 1, size) ||
    !isInCircle(x, y + 1, size)
  );
}

// --- Overlays ---
function distToSeg(px: number, py: number, x1: number, y1: number, x2: number, y2: number): number {
  const dx = x2 - x1, dy = y2 - y1;
  const lenSq = dx * dx + dy * dy;
  if (lenSq === 0) return Math.hypot(px - x1, py - y1);
  const t = Math.max(0, Math.min(1, ((px - x1) * dx + (py - y1) * dy) / lenSq));
  return Math.hypot(px - (x1 + t * dx), py - (y1 + t * dy));
}

function isCheckPixel(x: number, y: number, size: number, mirror?: boolean, flip?: boolean, offsetX = 0, offsetY = 0): boolean {
  const s = size * 0.3;
  const dentX = size / 2 + offsetX;
  const dentY = size * 0.75 + offsetY;
  // mirror flips for right eye, flip flips for animation alternation
  const dir = flip ? -1 : 1;
  // Short leg
  const d1 = distToSeg(x, y, dentX - dir * s * 0.35, dentY - s * 0.3, dentX, dentY);
  // Long leg
  const d2 = distToSeg(x, y, dentX, dentY, dentX + dir * s * 0.55, dentY - s * 0.9);
  return Math.min(d1, d2) < 1.5;
}

function isXPixel(x: number, y: number, size: number): boolean {
  const s = size * 0.42;
  const cx = size / 2, cy = size / 2;
  const d1 = distToSeg(x, y, cx - s, cy - s, cx + s, cy + s);
  const d2 = distToSeg(x, y, cx + s, cy - s, cx - s, cy + s);
  return Math.min(d1, d2) < 2.5;
}

function isTwirlPixel(x: number, y: number, size: number, angle: number): boolean {
  const cx = size / 2, cy = size / 2;
  const dx = x - cx, dy = y - cy;
  const dist = Math.hypot(dx, dy);
  if (dist < 0.5) return false;
  // Angle of this pixel from center
  const pixelAngle = Math.atan2(dy, dx);
  // Logarithmic spiral: combine angle with log of distance
  // Adding the rotation angle makes it spin
  const spiral = pixelAngle + Math.log(dist + 1) * 2.5 + angle;
  // Modulo to create alternating stripes
  const stripe = ((spiral / Math.PI) % 1 + 1) % 1;
  return stripe < 0.5;
}

// --- Lid math ---
interface LidResult {
  inCrescent: boolean;
  isLidBorder: boolean;
}

function computeLid(
  x: number, y: number,
  size: number,
  travel: number,
  angle: number,
  lidR: number,
  cutoffMult: number,
  fromTop: boolean,
): { inLid: boolean; inCut: boolean; lidCy: number; cutX: number; cutY: number } {
  const c = (size - 1) / 2;
  const r = size / 2;
  const startOffset = lidR + r;
  const cutDist = lidR * cutoffMult;

  const lidCy = fromTop
    ? c - startOffset + travel
    : c + startOffset - travel;
  const cutX = fromTop
    ? c + Math.sin(angle) * cutDist
    : c - Math.sin(angle) * cutDist;
  const cutY = fromTop
    ? lidCy + Math.cos(angle) * cutDist
    : lidCy - Math.cos(angle) * cutDist;

  const inLid = ((x - c) ** 2 + (y - lidCy) ** 2) <= lidR * lidR;
  const inCut = ((x - cutX) ** 2 + (y - cutY) ** 2) <= lidR * lidR;

  return { inLid, inCut, lidCy, cutX, cutY };
}

// --- Single Eye Renderer ---
function renderPixel(
  x: number, y: number,
  size: number,
  frame: EyeFrame,
  anim: EyeAnimation,
  tick: number,
  seed: number,
  px: number,
  mirror?: boolean,
): { bg: string } {
  const c = (size - 1) / 2;
  const r = size / 2;
  const lidR = r * anim.lidRadius;

  // Outside eye circle
  if (!isInCircle(x, y, size)) return { bg: "transparent" };

  // Compute lids
  const topLid = computeLid(x, y, size, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true);
  const botLid = computeLid(x, y, size, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false);

  const inTopCrescent = topLid.inLid && !topLid.inCut;
  const inBotCrescent = botLid.inLid && !botLid.inCut;

  const isCheck = frame.overlay === "check" || frame.overlay === "check-flip";
  const checkFlip = frame.overlay === "check-flip";

  // Check if lid covers this pixel (but don't erase checkmark pixels)
  if ((inTopCrescent || inBotCrescent)) {
    // Is this on the lid border? (neighbor inside eye but not in any crescent)
    const isLidBorder = [[-1,0],[1,0],[0,-1],[0,1]].some(([dx, dy]) => {
      const nx = x + dx, ny = y + dy;
      if (!isInCircle(nx, ny, size)) return false;
      const nTop = computeLid(nx, ny, size, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true);
      const nBot = computeLid(nx, ny, size, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false);
      return !(nTop.inLid && !nTop.inCut) && !(nBot.inLid && !nBot.inCut);
    });
    return { bg: isLidBorder ? ACCENT : "transparent" };
  }

  // Eye border - hidden where lid covers (but not checkmark pixels)
  if (isCircleBorder(x, y, size)) {
    if (frame.topLid > 0 || frame.bottomLid > 0) {
      const topCheck = computeLid(x, y, size, frame.topLid, frame.topLidAngle, lidR, anim.cutoffMult, true);
      const botCheck = computeLid(x, y, size, frame.bottomLid, frame.bottomLidAngle, lidR, anim.cutoffMult, false);
      if ((topCheck.inLid && !topCheck.inCut) || (botCheck.inLid && !botCheck.inCut)) {
        if (!(isCheck && isCheckPixel(x, y, size, mirror, checkFlip, frame.checkOffsetX ?? 0, frame.checkOffsetY ?? 0))) {
          return { bg: "transparent" };
        }
      }
    }
    return { bg: ACCENT };
  }

  // Pupil hole (rendered before overlays so owl-eye works)
  if (frame.pupilRatio > 0) {
    const pupilSize = Math.floor(size * frame.pupilRatio);
    const pupilCenter = Math.floor((size - pupilSize) / 2);
    const inPupil = isInCircle(x - pupilCenter - frame.pupilX, y - pupilCenter - frame.pupilY, pupilSize);
    if (inPupil) {
      // Check if this pupil pixel has a sparkle
      if (frame.sparkle) {
        const sc = (size - 1) / 2;
        const sx = Math.floor(sc + frame.pupilX - pupilSize * 0.25);
        const sy = Math.floor(sc + frame.pupilY - pupilSize * 0.25);
        const dx = x - sx, dy = y - sy;
        if (frame.sparkle === "diamond") {
          const phase = Math.floor(tick / 3) % 4;
          if (phase === 0 && dx === 0 && dy === 0) return { bg: ACCENT };
          if (phase === 1 && ((dx === 0 && Math.abs(dy) <= 1) || (dy === 0 && Math.abs(dx) <= 1))) return { bg: ACCENT };
          if (phase === 2 && dx === 0 && dy === 0) return { bg: ACCENT };
        }
      }
      // X inside pupil
      if (frame.overlay === "x") {
        // Gap ring: transparent border just inside pupil edge
        const gapOuter = pupilSize;
        const gapInner = pupilSize - 3;
        const distFromPupilCenter = Math.hypot(
          x - (pupilCenter + frame.pupilX + pupilSize / 2),
          y - (pupilCenter + frame.pupilY + pupilSize / 2)
        );
        if (distFromPupilCenter > gapInner / 2) {
          return { bg: "transparent" }; // gap ring
        }
        // Inside X: offset to follow pupil position
        if (isXPixel(x - frame.pupilX, y - frame.pupilY, size)) {
          const fastTick = tick * 5;
          let h2 = (x * 2654435761 + y * 2246822519 + (fastTick + seed) * 3266489917) >>> 0;
          h2 ^= h2 >> 16; h2 = (h2 * 2246822507) >>> 0; h2 ^= h2 >> 13; h2 = (h2 * 3266489909) >>> 0; h2 ^= h2 >> 16;
          return { bg: (h2 % 100) < 60 ? ACCENT : "transparent" };
        }
        // Inside pupil, outside X: transparent
        return { bg: "transparent" };
      }
      // Twirl inside pupil
      if (frame.overlay === "twirl") {
        const twirlAngle = frame.twirlAngle ?? 0;
        const isSpiralStripe = isTwirlPixel(x, y, size, twirlAngle);
        if (isSpiralStripe) {
          const cxs = size / 2, cys = size / 2;
          const dxs = x - cxs, dys = y - cys;
          const rx = Math.round(dxs * Math.cos(-twirlAngle) - dys * Math.sin(-twirlAngle) + cxs);
          const ry = Math.round(dxs * Math.sin(-twirlAngle) + dys * Math.cos(-twirlAngle) + cys);
          const slowTick = Math.floor(tick / 8);
          let h2 = (rx * 2654435761 + ry * 2246822519 + (slowTick + seed) * 3266489917) >>> 0;
          h2 ^= h2 >> 16; h2 = (h2 * 2246822507) >>> 0; h2 ^= h2 >> 13; h2 = (h2 * 3266489909) >>> 0; h2 ^= h2 >> 16;
          return { bg: (h2 % 100) < 80 ? ACCENT : "transparent" };
        }
      }
      return { bg: "transparent" };
    }
  }

  // Overlay (check)
  if (isCheck && isCheckPixel(x, y, size, mirror, checkFlip, frame.checkOffsetX ?? 0, frame.checkOffsetY ?? 0)) return { bg: "transparent" };
  // X overlay is now pupil-only, handled below in fill

  // Fill
  if (frame.solid) return { bg: ACCENT };
  // Error mode: 20% noise same speed for surrounding eye
  if (frame.overlay === "x") {
    let h = (x * 2654435761 + y * 2246822519 + (tick + seed) * 3266489917) >>> 0;
    h ^= h >> 16; h = (h * 2246822507) >>> 0; h ^= h >> 13; h = (h * 3266489909) >>> 0; h ^= h >> 16;
    return { bg: (h % 100) < 20 ? ACCENT : "transparent" };
  }
  // Twirl mode: sparse orange (20%) for surrounding eye
  if (frame.overlay === "twirl") {
    let h = (x * 2654435761 + y * 2246822519 + (tick + seed) * 3266489917) >>> 0;
    h ^= h >> 16; h = (h * 2246822507) >>> 0; h ^= h >> 13; h = (h * 3266489909) >>> 0; h ^= h >> 16;
    return { bg: (h % 100) < 20 ? ACCENT : "transparent" };
  }
  return { bg: noise(x, y, tick + seed) ? ACCENT : "transparent" };
}

function EyeRenderer({
  size, frame, anim, tick, seed, px, mirror,
}: {
  size: number;
  frame: EyeFrame;
  anim: EyeAnimation;
  tick: number;
  seed: number;
  px: number;
  mirror?: boolean;
}) {
  // Mirror: flip topLidAngle and pupilX for the right eye
  const effectiveFrame = mirror ? {
    ...frame,
    topLidAngle: -frame.topLidAngle,
  } : frame;

  return (
    <div className="inline-flex flex-col">
      {Array.from({ length: size }, (_, y) => (
        <div key={y} className="flex">
          {Array.from({ length: size }, (_, x) => {
            const { bg } = renderPixel(x, y, size, effectiveFrame, anim, tick, seed, px, mirror);
            return (
              <div key={x} style={{ width: px, height: px, background: bg }} />
            );
          })}
        </div>
      ))}
    </div>
  );
}

// --- Main component ---

export function WebMascot({
  mood = "idle",
  pixelSize = 5,
  debugFrame,
  debugAnim,
  onLoopComplete,
  onFrameUpdate,
}: {
  mood?: string;
  pixelSize?: number;
  debugFrame?: EyeFrame;
  debugAnim?: Partial<EyeAnimation>;
  onLoopComplete?: () => void;
  onFrameUpdate?: (frame: EyeFrame) => void;
}) {
  const baseAnim = EXPRESSIONS[mood] || EXPRESSIONS.idle;
  const anim = debugAnim ? { ...baseAnim, ...debugAnim } : baseAnim;

  const [tick, setTick] = useState(0);
  const [displayFrame, setDisplayFrame] = useState<EyeFrame>(anim.frames[0]);
  const frameIdxRef = useRef(0);
  const cancelledRef = useRef(false);
  const animRef = useRef(anim);
  const timersRef = useRef<number[]>([]);
  const onLoopRef = useRef(onLoopComplete);
  animRef.current = anim;
  onLoopRef.current = onLoopComplete;

  // Noise tick
  useEffect(() => {
    const interval = setInterval(() => setTick((t) => t + 1), anim.noiseSpeed);
    return () => clearInterval(interval);
  }, [anim.noiseSpeed]);

  // Animation loop using requestAnimationFrame for interpolation
  useEffect(() => {
    if (debugFrame) {
      setDisplayFrame(debugFrame);
      return;
    }
    if (anim.frames.length === 0) return;

    cancelledRef.current = false;
    frameIdxRef.current = 0;
    timersRef.current.forEach(clearTimeout);
    timersRef.current = [];

    function safeTimeout(fn: () => void, ms: number) {
      const id = window.setTimeout(fn, ms);
      timersRef.current.push(id);
      return id;
    }

    function scheduleNext() {
      if (cancelledRef.current) return;
      const a = animRef.current;
      const idx = frameIdxRef.current % a.frames.length;
      const f = a.frames[idx];
      if (!f) return;
      setDisplayFrame(f);

      const nextIdx = (idx + 1) % a.frames.length;
      const nextF = a.frames[nextIdx];
      const transition = f.transition;

      const holdTime = f.hold ?? a.frameDuration;

      if (transition && transition.duration > 0 && nextF) {
        safeTimeout(() => {
          if (cancelledRef.current) return;
          const startTime = performance.now();
          const dur = transition.duration;
          const easing = transition.easing;
          // Re-read the frames at interpolation time too
          const fromFrame = animRef.current.frames[idx] || f;
          const toFrame = animRef.current.frames[nextIdx] || nextF;

          function step() {
            if (cancelledRef.current) return;
            const elapsed = performance.now() - startTime;
            const rawT = Math.min(elapsed / dur, 1);
            const easedT = applyEasing(rawT, easing);
            setDisplayFrame(interpolateFrames(fromFrame, toFrame, easedT));

            if (rawT < 1) {
              requestAnimationFrame(step);
            } else {
              frameIdxRef.current = nextIdx;
              if (nextIdx === 0) {
                onLoopRef.current?.();
                if (!animRef.current.loop) return;
              }
              scheduleNext();
            }
          }
          requestAnimationFrame(step);
        }, holdTime);
      } else {
        safeTimeout(() => {
          if (cancelledRef.current) return;
          frameIdxRef.current = nextIdx;
          if (nextIdx === 0) {
            onLoopRef.current?.();
            if (!animRef.current.loop) return;
          }
          scheduleNext();
        }, holdTime);
      }
    }

    scheduleNext();
    return () => {
      cancelledRef.current = true;
      timersRef.current.forEach(clearTimeout);
      timersRef.current = [];
    };
  }, [debugFrame, anim.frames.length, anim.frameDuration, anim.loop]); // eslint-disable-line

  const currentFrame = debugFrame || displayFrame;

  // Report current frame to parent
  const onFrameRef = useRef(onFrameUpdate);
  onFrameRef.current = onFrameUpdate;
  useEffect(() => {
    onFrameRef.current?.(currentFrame);
  }, [currentFrame]);
  const px = pixelSize;

  return (
    <div className="inline-flex flex-col">
      <div className="flex items-center" style={{ gap: anim.gap * px }}>
        <EyeRenderer size={anim.eyeSize} frame={currentFrame} anim={anim} tick={tick} seed={0} px={px} mirror={false} />
        <EyeRenderer size={anim.eyeSize} frame={currentFrame} anim={anim} tick={tick} seed={7777} px={px} mirror={true} />
      </div>
    </div>
  );
}

// --- Hero logo: cycles through all expressions with scramble blink transitions ---
const HERO_MOODS = ["idle", "thinking", "asking", "done", "error"];
const HERO_LABELS: Record<string, string> = {
  idle: "idle",
  thinking: "thinking...",
  asking: "asking...",
  done: "done",
  error: "error",
};
const HERO_SHOW_MS = 5000;
const HERO_DISSOLVE_MS = 500;

export function MascotLogo() {
  const [moodIdx, setMoodIdx] = useState(0);
  const [frozen, setFrozen] = useState(false);
  const frozenRef = useRef(false);

  const currentMood = HERO_MOODS[moodIdx];
  const anim = EXPRESSIONS[currentMood] || EXPRESSIONS.idle;

  const doSwap = useCallback(() => {
    if (frozenRef.current) return;
    frozenRef.current = true;
    setFrozen(true);
    setTimeout(() => {
      setMoodIdx((i) => (i + 1) % HERO_MOODS.length);
      setFrozen(false);
      frozenRef.current = false;
    }, 400);
  }, []);

  // Swap either on loop complete OR after max time, whichever first
  const handleLoopComplete = useCallback(() => doSwap(), [doSwap]);

  useEffect(() => {
    frozenRef.current = false;
    const timer = setTimeout(() => doSwap(), HERO_SHOW_MS);
    return () => clearTimeout(timer);
  }, [moodIdx, doSwap]);

  return (
    <div className="flex flex-col items-center gap-3">
      {frozen ? (
        <div className="inline-flex flex-col">
          <div className="flex items-center" style={{ gap: anim.gap * 6 }}>
            <ClosedEyeCircle size={anim.eyeSize} px={6} />
            <ClosedEyeCircle size={anim.eyeSize} px={6} />
          </div>
        </div>
      ) : (
        <WebMascot mood={currentMood} pixelSize={6} onLoopComplete={handleLoopComplete} />
      )}
      <span className="font-mono text-sm text-accent tracking-wider">
        {HERO_LABELS[currentMood]}
      </span>
    </div>
  );
}

// Fully closed eye: just the circle outline, everything inside transparent
function ClosedEyeCircle({ size, px }: { size: number; px: number }) {
  return (
    <div className="inline-flex flex-col">
      {Array.from({ length: size }, (_, y) => (
        <div key={y} className="flex">
          {Array.from({ length: size }, (_, x) => (
            <div key={x} style={{ width: px, height: px, background: "transparent" }} />
          ))}
        </div>
      ))}
    </div>
  );
}

// --- Status to mood ---
export function statusToMood(status: string): string {
  switch (status) {
    case "thinking": return "thinking";
    case "asking": return "asking";
    case "executing": return "thinking";
    case "done": return "done";
    case "error": return "error";
    default: return "idle";
  }
}
