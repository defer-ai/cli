"use client";

import { useState, useEffect } from "react";
import { WebMascot } from "../mascot";
import { EXPRESSIONS, type EyeFrame, type EyeAnimation, EASE } from "../eye-animation";

function Slider({ label, value, onChange, min = 0, max = 60, step = 0.5 }: {
  label: string; value: number; onChange: (v: number) => void;
  min?: number; max?: number; step?: number;
}) {
  const display = Number.isInteger(value) ? value : value.toFixed(2);
  return (
    <div>
      <label className="text-gray-500 text-[11px] block">{label}: {display}</label>
      <input type="range" min={min} max={max} step={step} value={value}
        onChange={(e) => onChange(Number(e.target.value))} className="w-full h-4" />
    </div>
  );
}

function FrameEditor({ frame, onChange }: { frame: EyeFrame; onChange: (f: EyeFrame) => void }) {
  const set = (key: keyof EyeFrame, val: number) => onChange({ ...frame, [key]: val });
  const trans = frame.transition || { duration: 0, easing: EASE.linear };
  const setTrans = (dur: number, attack: number, decay: number) => {
    onChange({
      ...frame,
      transition: dur > 0 ? { duration: dur, easing: { attack, decay } } : undefined,
    });
  };

  return (
    <div className="space-y-0.5">
      <Slider label="pupil ratio" value={frame.pupilRatio} onChange={(v) => set("pupilRatio", v)} min={0} max={1} step={0.05} />
      <Slider label="pupil X" value={frame.pupilX} onChange={(v) => set("pupilX", v)} min={-5} max={5} step={0.5} />
      <Slider label="pupil Y" value={frame.pupilY} onChange={(v) => set("pupilY", v)} min={-5} max={5} step={0.5} />
      <Slider label="top lid" value={frame.topLid} onChange={(v) => set("topLid", v)} min={0} max={60} />
      <Slider label="top lid angle" value={frame.topLidAngle} onChange={(v) => set("topLidAngle", v)} min={-3.14} max={3.14} step={0.05} />
      <Slider label="bottom lid" value={frame.bottomLid} onChange={(v) => set("bottomLid", v)} min={0} max={60} />
      <Slider label="bottom lid angle" value={frame.bottomLidAngle} onChange={(v) => set("bottomLidAngle", v)} min={-3.14} max={3.14} step={0.05} />
      <Slider label="check X" value={frame.checkOffsetX ?? 0} onChange={(v) => onChange({ ...frame, checkOffsetX: v })} min={-10} max={10} step={0.5} />
      <Slider label="check Y" value={frame.checkOffsetY ?? 0} onChange={(v) => onChange({ ...frame, checkOffsetY: v })} min={-10} max={10} step={0.5} />
      <p className="text-gray-600 text-[10px] mt-2">Timing</p>
      <Slider label="hold ms" value={frame.hold ?? 0} onChange={(v) => onChange({ ...frame, hold: v > 0 ? v : undefined })} min={0} max={3000} step={50} />
      <Slider label="duration ms" value={trans.duration} onChange={(v) => setTrans(v, trans.easing.attack, trans.easing.decay)} min={0} max={2000} step={50} />
      {trans.duration > 0 && (
        <>
          <Slider label="attack" value={trans.easing.attack} onChange={(v) => setTrans(trans.duration, v, trans.easing.decay)} min={0} max={1} step={0.05} />
          <Slider label="decay" value={trans.easing.decay} onChange={(v) => setTrans(trans.duration, trans.easing.attack, v)} min={0} max={1} step={0.05} />
        </>
      )}
    </div>
  );
}

function ExpressionSection({ name, anim }: { name: string; anim: EyeAnimation }) {
  const [frames, setFrames] = useState<EyeFrame[]>(anim.frames);
  const [selectedFrame, setSelectedFrame] = useState(0);
  const [eyeSize, setEyeSize] = useState(anim.eyeSize);
  const [lidRadius, setLidRadius] = useState(anim.lidRadius);
  const [cutoffMult, setCutoffMult] = useState(anim.cutoffMult);
  const [frameDuration, setFrameDuration] = useState(anim.frameDuration);
  const [noiseSpeed, setNoiseSpeed] = useState(anim.noiseSpeed);
  const [copied, setCopied] = useState(false);
  const [loaded, setLoaded] = useState(false);

  // Load from localStorage after mount (avoids hydration mismatch)
  useEffect(() => {
    try {
      const raw = localStorage.getItem(`mascot-${name}`);
      if (raw) {
        const saved = JSON.parse(raw);
        if (saved.frames) setFrames(saved.frames);
        if (saved.eyeSize) setEyeSize(saved.eyeSize);
        if (saved.lidRadius) setLidRadius(saved.lidRadius);
        if (saved.cutoffMult) setCutoffMult(saved.cutoffMult);
        if (saved.frameDuration) setFrameDuration(saved.frameDuration);
        if (saved.noiseSpeed) setNoiseSpeed(saved.noiseSpeed);
      }
    } catch {}
    setLoaded(true);
  }, [name]);

  // Auto-save to localStorage on every change (only after initial load)
  useEffect(() => {
    if (!loaded) return;
    const data = { frames, eyeSize, lidRadius, cutoffMult, frameDuration, noiseSpeed };
    localStorage.setItem(`mascot-${name}`, JSON.stringify(data));
  }, [frames, eyeSize, lidRadius, cutoffMult, frameDuration, noiseSpeed, name, loaded]);

  const updateFrame = (idx: number, f: EyeFrame) => {
    const next = [...frames];
    next[idx] = f;
    setFrames(next);
  };

  const addFrame = () => {
    const last = frames[frames.length - 1] || {
      pupilRatio: 0.65, pupilX: 0, pupilY: 0,
      topLid: 0, topLidAngle: 0, bottomLid: 0, bottomLidAngle: 0,
    };
    setFrames([...frames, { ...last }]);
    setSelectedFrame(frames.length);
  };

  const removeFrame = (idx: number) => {
    if (frames.length <= 1) return;
    const next = frames.filter((_, i) => i !== idx);
    setFrames(next);
    if (selectedFrame >= next.length) setSelectedFrame(next.length - 1);
  };

  const duplicateFrame = (idx: number) => {
    const next = [...frames];
    next.splice(idx + 1, 0, { ...frames[idx] });
    setFrames(next);
    setSelectedFrame(idx + 1);
  };

  const customAnim: Partial<EyeAnimation> = {
    eyeSize, lidRadius, cutoffMult, frameDuration, noiseSpeed,
    frames,
  };

  return (
    <div className="border border-gray-800 rounded-lg p-4">
      <h3 className="text-orange-500 font-bold text-lg mb-4">{name}</h3>

      <div className="flex gap-6">
        {/* Left: animated result + global controls */}
        <div className="flex flex-col items-center gap-3 min-w-[180px]">
          <p className="text-gray-600 text-[10px]">Animated result</p>
          <WebMascot mood={name} pixelSize={4} debugAnim={customAnim} />

          <div className="w-full space-y-0.5 mt-2">
            <Slider label="eye size" value={eyeSize} onChange={setEyeSize} min={10} max={40} step={1} />
            <Slider label="lid radius" value={lidRadius} onChange={setLidRadius} min={1} max={5} step={0.1} />
            <Slider label="cutoff" value={cutoffMult} onChange={setCutoffMult} min={0.3} max={2} step={0.05} />
            <Slider label="frame ms" value={frameDuration} onChange={setFrameDuration} min={50} max={2000} step={50} />
            <Slider label="noise ms" value={noiseSpeed} onChange={setNoiseSpeed} min={20} max={200} step={10} />
          </div>
        </div>

        {/* Middle: frame timeline */}
        <div className="flex flex-col gap-2 min-w-[280px]">
          <div className="flex items-center gap-2">
            <p className="text-gray-600 text-[10px]">Frames ({frames.length})</p>
            <button onClick={addFrame} className="text-[10px] text-orange-500 hover:text-orange-300 cursor-pointer">+ add</button>
          </div>

          <div className="flex flex-wrap gap-1">
            {frames.map((f, i) => (
              <button
                key={i}
                onClick={() => setSelectedFrame(i)}
                className={`w-8 h-8 text-[10px] rounded cursor-pointer ${
                  i === selectedFrame
                    ? "bg-orange-500 text-black font-bold"
                    : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                }`}
              >
                {i}
              </button>
            ))}
          </div>

          {/* Selected frame preview + controls */}
          {frames[selectedFrame] && (
            <div className="flex gap-4 mt-2">
              <div className="flex flex-col items-center gap-1">
                <WebMascot
                  mood={name}
                  pixelSize={4}
                  debugFrame={frames[selectedFrame]}
                  debugAnim={customAnim}
                />
                <p className="text-gray-600 text-[10px]">frame {selectedFrame}</p>
                <div className="flex gap-2">
                  <button
                    onClick={() => duplicateFrame(selectedFrame)}
                    className="text-[10px] text-gray-500 hover:text-white cursor-pointer"
                  >
                    dup
                  </button>
                  <button
                    onClick={() => removeFrame(selectedFrame)}
                    className="text-[10px] text-red-500 hover:text-red-300 cursor-pointer"
                  >
                    del
                  </button>
                </div>
              </div>

              <div className="min-w-[200px]">
                <FrameEditor
                  frame={frames[selectedFrame]}
                  onChange={(f) => updateFrame(selectedFrame, f)}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* JSON export + actions */}
      <div className="mt-3 flex gap-3 items-center">
        <button
          onClick={() => {
            navigator.clipboard.writeText(JSON.stringify({ ...customAnim, name }, null, 2));
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
          }}
          className="text-[10px] px-2 py-1 bg-orange-500/20 text-orange-500 rounded cursor-pointer hover:bg-orange-500/30"
        >
          {copied ? "Copied!" : "Copy JSON"}
        </button>
        <button
          onClick={() => {
            setFrames(anim.frames);
            setEyeSize(anim.eyeSize);
            setLidRadius(anim.lidRadius);
            setCutoffMult(anim.cutoffMult);
            setFrameDuration(anim.frameDuration);
            setNoiseSpeed(anim.noiseSpeed);
            localStorage.removeItem(`mascot-${name}`);
          }}
          className="text-[10px] px-2 py-1 bg-gray-800 text-gray-500 rounded cursor-pointer hover:bg-gray-700 hover:text-gray-300"
        >
          Reset to default
        </button>
      </div>
      <details className="mt-2">
        <summary className="text-gray-600 text-[10px] cursor-pointer hover:text-gray-400">Show JSON</summary>
        <pre className="text-orange-500/60 font-mono text-[9px] mt-1 max-h-40 overflow-auto bg-black/30 p-2 rounded">
          {JSON.stringify({ ...customAnim, name }, null, 2)}
        </pre>
      </details>
    </div>
  );
}

export default function MascotPreview() {
  const expressions = Object.entries(EXPRESSIONS).filter(([k]) => !k.startsWith("test"));

  return (
    <div className="min-h-screen bg-[#0a0a0a] p-8">
      <h1 className="text-white text-2xl font-bold mb-8">Mascot Expression Editor</h1>

      <div className="space-y-6">
        {expressions.map(([name, anim]) => (
          <ExpressionSection key={name} name={name} anim={anim} />
        ))}
      </div>
    </div>
  );
}
