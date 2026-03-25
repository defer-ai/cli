"use client";

import { useState, useEffect } from "react";

type Mood = "idle" | "thinking" | "asking" | "done";

// Faithful recreation of the CLI mascot as a pixel grid
// The mascot is: two rectangular eyes (6w x 3-4h) with a gap, and a centered mouth bar
// 0 = transparent, 1 = cyan (border), 2 = white (eye interior), 3 = dark (pupil), 4 = sparkle

// Each frame is a 2D grid representing the full mascot
type Frame = number[][];

// Fixed height: all frames are exactly MAX_EYE_H + 4 rows (top border, eye rows, bottom border, gap, mouth)
const MAX_EYE_H = 2;
const FRAME_ROWS = MAX_EYE_H + 4; // top + 2 eye rows + bottom + gap + mouth

function makeFrame(
  eyeHeight: number,
  leftPupil: number[][],
  rightPupil: number[][],
  mouthWidth: number
): Frame {
  const W = 21;
  const eyeW = 6;
  const gap = 3;
  const eyeStartL = 1;
  const eyeStartR = eyeStartL + eyeW + gap;
  const mouthStart = Math.floor((W - mouthWidth) / 2);

  const rows: Frame = [];

  // Eye top border
  const topRow = new Array(W).fill(0);
  for (let x = eyeStartL + 1; x < eyeStartL + eyeW - 1; x++) topRow[x] = 1;
  for (let x = eyeStartR + 1; x < eyeStartR + eyeW - 1; x++) topRow[x] = 1;
  rows.push([...topRow]);

  // Eye body rows (always MAX_EYE_H, pad with white if fewer)
  for (let y = 0; y < MAX_EYE_H; y++) {
    const row = new Array(W).fill(0);
    // Left eye
    row[eyeStartL] = 1;
    row[eyeStartL + eyeW - 1] = 1;
    for (let x = 1; x < eyeW - 1; x++) {
      row[eyeStartL + x] = y < eyeHeight ? (leftPupil[y]?.[x - 1] ?? 2) : 2;
    }
    // Right eye
    row[eyeStartR] = 1;
    row[eyeStartR + eyeW - 1] = 1;
    for (let x = 1; x < eyeW - 1; x++) {
      row[eyeStartR + x] = y < eyeHeight ? (rightPupil[y]?.[x - 1] ?? 2) : 2;
    }
    rows.push(row);
  }

  // Eye bottom border
  const botRow = new Array(W).fill(0);
  for (let x = eyeStartL + 1; x < eyeStartL + eyeW - 1; x++) botRow[x] = 1;
  for (let x = eyeStartR + 1; x < eyeStartR + eyeW - 1; x++) botRow[x] = 1;
  rows.push([...botRow]);

  // Gap between eyes and mouth
  rows.push(new Array(W).fill(0));

  // Mouth
  const mouthRow = new Array(W).fill(0);
  for (let x = mouthStart; x < mouthStart + mouthWidth; x++) mouthRow[x] = 1;
  rows.push(mouthRow);

  return rows;
}

// Pupil patterns (4 wide for the interior of each eye)
// 2 = white, 3 = dark pupil, 4 = sparkle

const PUPIL_EMPTY: number[][] = [
  [2, 2, 2, 2],
  [2, 2, 2, 2],
];
const PUPIL_HALF: number[][] = [
  [2, 3, 3, 2],
  [2, 2, 2, 2],
];
const PUPIL_CLOSED: number[][] = [
  [3, 3, 3, 3],
];
const PUPIL_SNAKE_1: number[][] = [
  [2, 2, 3, 2],
  [2, 3, 2, 2],
];
const PUPIL_SNAKE_2: number[][] = [
  [2, 3, 2, 2],
  [2, 2, 3, 2],
];
const PUPIL_BIG: number[][] = [
  [2, 3, 3, 2],
  [2, 3, 3, 2],
];
const PUPIL_QUESTION_1: number[][] = [
  [2, 3, 3, 2],
  [2, 2, 3, 2],
];
const PUPIL_QUESTION_2: number[][] = [
  [2, 2, 2, 2],
  [2, 2, 3, 2],
];
const PUPIL_SPARKLE_1: number[][] = [
  [2, 4, 2, 2],
  [2, 2, 2, 2],
];
const PUPIL_SPARKLE_2: number[][] = [
  [2, 2, 2, 4],
  [2, 2, 2, 2],
];
const PUPIL_SQUINT: number[][] = [
  [2, 3, 3, 2],
];

const FRAMES: Record<Mood, Frame[]> = {
  idle: [
    makeFrame(2, PUPIL_EMPTY, PUPIL_EMPTY, 8),
    makeFrame(2, PUPIL_HALF, PUPIL_HALF, 8),
    makeFrame(1, PUPIL_CLOSED, PUPIL_CLOSED, 8),
    makeFrame(2, PUPIL_HALF, PUPIL_HALF, 8),
  ],
  thinking: [
    makeFrame(2, PUPIL_SNAKE_1, PUPIL_SNAKE_1, 8),
    makeFrame(2, PUPIL_SNAKE_2, PUPIL_SNAKE_2, 8),
  ],
  asking: [
    makeFrame(2, PUPIL_BIG, PUPIL_BIG, 8),
    makeFrame(2, PUPIL_QUESTION_1, PUPIL_QUESTION_1, 8),
    makeFrame(2, PUPIL_QUESTION_2, PUPIL_QUESTION_2, 8),
  ],
  done: [
    makeFrame(1, PUPIL_SQUINT, PUPIL_SQUINT, 8),
    makeFrame(2, PUPIL_SPARKLE_1, PUPIL_SPARKLE_2, 8),
    makeFrame(2, PUPIL_SPARKLE_2, PUPIL_SPARKLE_1, 8),
  ],
};

const SPEEDS: Record<Mood, number> = {
  idle: 600,
  thinking: 200,
  asking: 500,
  done: 500,
};

const PIXEL_COLORS: Record<number, string> = {
  0: "",                             // transparent
  1: "bg-cyan-400",                  // border / mouth
  2: "bg-gray-100 dark:bg-gray-200", // eye white
  3: "bg-cyan-800 dark:bg-cyan-900", // pupil
  4: "bg-yellow-300",                // sparkle
};

export function WebMascot({
  mood,
  pixelSize = 4,
  speed,
}: {
  mood: Mood;
  pixelSize?: number;
  speed?: number;
}) {
  const [frame, setFrame] = useState(0);
  const frames = FRAMES[mood];
  const animSpeed = speed ?? SPEEDS[mood];

  useEffect(() => {
    setFrame(0);
    const interval = setInterval(() => {
      setFrame((f) => (f + 1) % frames.length);
    }, animSpeed);
    return () => clearInterval(interval);
  }, [mood, frames.length, animSpeed]);

  const grid = frames[frame % frames.length];
  const px = pixelSize;

  return (
    <div className="inline-flex flex-col">
      {grid.map((row, y) => (
        <div key={y} className="flex">
          {row.map((cell, x) => (
            <div
              key={x}
              className={PIXEL_COLORS[cell] || ""}
              style={{
                width: px,
                height: px,
                ...(cell === 0 ? { background: "transparent" } : {}),
              }}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

/** Logo mascot: noise eyes + "defer.sh" as the mouth */
export function MascotLogo() {
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => setFrame((f) => f + 1), 80);
    return () => clearInterval(interval);
  }, []);

  const px = 35;
  const eyeW = 6;
  const eyeH = MAX_EYE_H;

  const noiseColors = [
    "bg-black",
    "bg-white",
  ];

  function NoiseEye() {
    const rows = [];
    // Top border
    rows.push(
      <div key="top" className="flex">
        <div style={{ width: px, height: px }} />
        {Array.from({ length: eyeW - 2 }, (_, i) => (
          <div key={i} className="bg-cyan-400" style={{ width: px, height: px }} />
        ))}
        <div style={{ width: px, height: px }} />
      </div>
    );
    // Body
    for (let y = 0; y < eyeH; y++) {
      rows.push(
        <div key={`body-${y}`} className="flex">
          <div className="bg-cyan-400" style={{ width: px, height: px }} />
          {Array.from({ length: eyeW - 2 }, (_, x) => {
            const colorIdx = (frame + y * 7 + x * 3) % noiseColors.length;
            return (
              <div
                key={x}
                className={noiseColors[colorIdx]}
                style={{ width: px, height: px }}
              />
            );
          })}
          <div className="bg-cyan-400" style={{ width: px, height: px }} />
        </div>
      );
    }
    // Bottom border
    rows.push(
      <div key="bot" className="flex">
        <div style={{ width: px, height: px }} />
        {Array.from({ length: eyeW - 2 }, (_, i) => (
          <div key={i} className="bg-cyan-400" style={{ width: px, height: px }} />
        ))}
        <div style={{ width: px, height: px }} />
      </div>
    );
    return <div className="inline-flex flex-col">{rows}</div>;
  }

  return (
    <div className="flex flex-col items-center gap-4">
      {/* Eyes */}
      <div className="flex gap-10">
        <NoiseEye />
        <NoiseEye />
      </div>
      {/* Mouth = defer.sh */}
      <span className="font-mono text-2xl text-accent tracking-[0.3em]">
        defer.sh
      </span>
    </div>
  );
}
