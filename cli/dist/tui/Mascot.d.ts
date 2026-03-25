export type MascotMood = "idle" | "thinking" | "asking" | "answering" | "executing" | "done" | "error";
export declare function Mascot({ mood }: {
    mood: MascotMood;
}): import("react/jsx-runtime").JSX.Element;
/** Map agent status to mascot mood */
export declare function statusToMood(status: string, phase?: string): MascotMood;
