export interface ParsedOption {
    label: string;
    value: string;
}
interface Props {
    options?: ParsedOption[];
    onSubmit: (value: string) => void;
    onCancel: () => void;
}
export declare function InputBar({ options, onSubmit, onCancel }: Props): import("react/jsx-runtime").JSX.Element;
export {};
