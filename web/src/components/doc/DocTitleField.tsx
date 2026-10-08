import type { RefObject } from "react";

import { pageTitleClass } from "@/components/PageHeader";
import { TitleTextarea } from "@/components/TitleTextarea";
import { cn } from "@/lib/utils";

interface DocTitleFieldProps {
  editable: boolean;
  title: string;
  staticTitle: string;
  onChange: (value: string) => void;
  onBlur?: () => void;
  inputRef?: RefObject<HTMLTextAreaElement | null>;
}

export const DocTitleField = ({ editable, title, staticTitle, onChange, onBlur, inputRef }: DocTitleFieldProps) => (
  <>
    {!editable && (
      <h1 dir="auto" className={pageTitleClass}>
        {staticTitle}
      </h1>
    )}
    {editable && (
      <TitleTextarea
        ref={inputRef}
        dir="auto"
        value={title}
        onValueChange={onChange}
        onBlur={onBlur}
        blurOnEnter
        aria-label="Title"
        className={cn(pageTitleClass, "text-start")}
        data-testid="doc-title-input"
      />
    )}
  </>
);
