"use client";

import { useState, useEffect } from "react";

type Mood = "idle" | "thinking" | "asking" | "done";

// CSS pixel grid mascot - each eye is a 4x3 grid, mouth is a row
// 1 = filled (eye border), 0 = empty (white of eye), 2 = pupil
type Grid = number[][];

interface Face {
  leftEye: Grid;
  rightEye: Grid;
  mouth: Grid;
}

const FACES: Record<Mood, Face[]> = {
  idle: [
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 1, 1, 1, 1, 0]],
    },
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 1, 1, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 1, 1, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 1, 1, 1, 1, 0]],
    },
  ],
  thinking: [
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 0, 2, 1],
        [1, 2, 0, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 0, 2, 1],
        [1, 2, 0, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 0, 1, 1, 0, 0]],
    },
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 2, 0, 1],
        [1, 0, 2, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 2, 0, 1],
        [1, 0, 2, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 0, 1, 1, 0, 0]],
    },
  ],
  asking: [
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 2, 2, 1],
        [1, 2, 2, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 2, 2, 1],
        [1, 2, 2, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 1, 1, 1, 1, 0]],
    },
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 0, 2, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 0, 2, 1],
        [1, 1, 1, 1],
      ],
      mouth: [[0, 1, 1, 1, 1, 0]],
    },
  ],
  done: [
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 0, 0, 1],
        [1, 1, 1, 1],
      ],
      mouth: [
        [0, 1, 1, 1, 1, 0],
        [0, 0, 1, 1, 0, 0],
      ],
    },
    {
      leftEye: [
        [1, 1, 1, 1],
        [1, 3, 0, 1],
        [1, 1, 1, 1],
      ],
      rightEye: [
        [1, 1, 1, 1],
        [1, 0, 3, 1],
        [1, 1, 1, 1],
      ],
      mouth: [
        [0, 1, 1, 1, 1, 0],
        [0, 0, 1, 1, 0, 0],
      ],
    },
  ],
};

const COLORS: Record<number, string> = {
  0: "bg-white",        // white of eye
  1: "bg-cyan-400",     // border/fill
  2: "bg-cyan-900",     // pupil
  3: "bg-yellow-300",   // sparkle
};

function PixelGrid({
  grid,
  pixelSize = 4,
}: {
  grid: Grid;
  pixelSize?: number;
}) {
  return (
    <div className="inline-flex flex-col">
      {grid.map((row, y) => (
        <div key={y} className="flex">
          {row.map((cell, x) => (
            <div
              key={x}
              className={`${COLORS[cell] || "bg-transparent"}`}
              style={{ width: pixelSize, height: pixelSize }}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

export function WebMascot({
  mood,
  pixelSize = 4,
  speed = 600,
}: {
  mood: Mood;
  pixelSize?: number;
  speed?: number;
}) {
  const [frame, setFrame] = useState(0);
  const frames = FACES[mood];

  useEffect(() => {
    setFrame(0);
    const interval = setInterval(() => {
      setFrame((f) => (f + 1) % frames.length);
    }, speed);
    return () => clearInterval(interval);
  }, [mood, frames.length, speed]);

  const face = frames[frame % frames.length];
  const gap = pixelSize * 3;

  return (
    <div className="inline-flex flex-col items-center" style={{ gap: pixelSize * 2 }}>
      {/* Eyes */}
      <div className="flex items-end" style={{ gap }}>
        <PixelGrid grid={face.leftEye} pixelSize={pixelSize} />
        <PixelGrid grid={face.rightEye} pixelSize={pixelSize} />
      </div>
      {/* Mouth */}
      <PixelGrid grid={face.mouth} pixelSize={pixelSize} />
    </div>
  );
}

/** Hero mascot - larger version */
export function HeroMascot() {
  const [mood, setMood] = useState<Mood>("idle");

  // Cycle through moods
  useEffect(() => {
    const moods: Mood[] = ["idle", "thinking", "asking", "done"];
    let idx = 0;
    const interval = setInterval(() => {
      idx = (idx + 1) % moods.length;
      setMood(moods[idx]);
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  return <WebMascot mood={mood} pixelSize={6} speed={400} />;
}
