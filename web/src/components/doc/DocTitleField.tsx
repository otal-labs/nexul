import type { RefObject } from "react";

import { Input } from "@/components/ui/input";

interface DocTitleFieldProps {
  editable: boolean;
  title: string;
  staticTitle: string;
  onChange: (value: string) => void;
  onBlur: () => void;
  inputRef: RefObject<HTMLInputElement | null>;
}

export const DocTitleField = ({ editable, title, staticTitle, onChange, onBlur, inputRef }: DocTitleFieldProps) => (
  <>
    {!editable && <h1 className="mt-3 text-4xl font-semibold tracking-tight sm:text-5xl">{staticTitle}</h1>}
    {editable && (
      <Input
        ref={inputRef}
        value={title}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key !== "Enter") return;
          e.preventDefault();
          e.currentTarget.blur();
        }}
        onBlur={onBlur}
        aria-label="Title"
        className="mt-3 h-auto w-full border-0 bg-transparent px-0 py-0 text-4xl font-semibold tracking-tight focus-visible:ring-0 sm:text-5xl"
        data-testid="doc-title-input"
      />
    )}
  </>
);
