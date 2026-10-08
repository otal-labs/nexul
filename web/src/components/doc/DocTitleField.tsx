import type { RefObject } from "react";

import { TitleTextarea } from "@/components/TitleTextarea";

interface DocTitleFieldProps {
  editable: boolean;
  title: string;
  staticTitle: string;
  onChange: (value: string) => void;
  onBlur: () => void;
  inputRef: RefObject<HTMLTextAreaElement | null>;
}

export const DocTitleField = ({ editable, title, staticTitle, onChange, onBlur, inputRef }: DocTitleFieldProps) => (
  <>
    {!editable && <h1 className="mt-3 text-center text-4xl font-semibold tracking-tight sm:text-5xl">{staticTitle}</h1>}
    {editable && (
      <TitleTextarea
        ref={inputRef}
        value={title}
        onValueChange={onChange}
        onBlur={onBlur}
        blurOnEnter
        aria-label="Title"
        className="mt-3 text-4xl sm:text-5xl"
        data-testid="doc-title-input"
      />
    )}
  </>
);
