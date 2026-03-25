import type { LLMProvider } from "../providers/types.js";
interface AppProps {
    task: string;
    provider: LLMProvider;
}
export declare function App({ task, provider }: AppProps): import("react/jsx-runtime").JSX.Element;
export {};
