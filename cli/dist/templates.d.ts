export type Target = "claude-code" | "cursor" | "chatgpt" | "universal" | "api";
export interface Template {
    filename: string;
    description: string;
    content: string;
}
export declare const templates: Record<Target, Template>;
