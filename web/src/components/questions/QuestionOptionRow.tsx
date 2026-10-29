import type { ReactNode } from "react";

import type { QuestionOption } from "@nexul/client-core/question";
import { cn } from "@/lib/utils";

interface QuestionOptionRowProps {
  number: number;
  option: QuestionOption;
  checked: boolean;
  // control is the radio or checkbox the step chose; the row only lays it out beside the number and the label.
  control: ReactNode;
  htmlFor: string;
}

// One hairline option row: its number key, the control, the label, and the description muted beneath.
export const QuestionOptionRow = ({ number, option, checked, control, htmlFor }: QuestionOptionRowProps) => (
  <li className={cn("transition-colors duration-150 ease-standard hover:bg-accent/40", checked && "bg-accent/40")}>
    <label htmlFor={htmlFor} className="flex cursor-pointer items-start gap-2.5 px-3 py-2">
      <span className="w-3 shrink-0 font-mono text-[10px] leading-5 text-muted-foreground tabular-nums">{number}</span>
      <span className="flex h-5 shrink-0 items-center">{control}</span>
      <span className="min-w-0 flex-1">
        <span className="block text-sm">{option.label}</span>
        {option.description && <span className="block text-xs text-muted-foreground">{option.description}</span>}
      </span>
    </label>
  </li>
);
