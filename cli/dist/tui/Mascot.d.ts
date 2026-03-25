export type MascotMood = "idle" | "thinking" | "asking" | "answering" | "executing" | "done" | "error";
export declare function Mascot({ mood }: {
    mood: MascotMood;
}): import("react/jsx-runtime").JSX.Element;
export declare function MiniMascot({ mood }: {
    mood: MascotMood;
}): import("react/jsx-runtime").JSX.Element;
export declare function statusToMood(status: string, _phase?: string): MascotMood;
