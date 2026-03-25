interface LogOptions {
    category?: string;
    question?: string;
    answer?: string;
    delegated?: boolean;
}
export declare function logCommand(options: LogOptions): Promise<void>;
export {};
