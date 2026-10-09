import type { ComponentProps } from "react";

import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface TitleTextareaProps extends Omit<ComponentProps<"textarea">, "onChange"> {
  onValueChange: (value: string) => void;
  blurOnEnter?: boolean;
}

// A title is one logical line that wraps visually: Enter never inserts a break and pasted breaks become spaces.
export const TitleTextarea = ({ onValueChange, blurOnEnter = false, className, onKeyDown, ...props }: TitleTextareaProps) => (
  <Textarea
    rows={1}
    onChange={(e) => onValueChange(e.target.value.replace(/\s*[\r\n]+\s*/g, " "))}
    onKeyDown={(e) => {
      onKeyDown?.(e);
      if (e.key !== "Enter" || e.nativeEvent.isComposing) return;
      e.preventDefault();
      if (blurOnEnter) e.currentTarget.blur();
    }}
    className={cn(
      "field-sizing-content min-h-0 resize-none rounded-none border-0 bg-transparent p-0 text-left font-semibold tracking-tight text-balance shadow-none focus-visible:ring-0 dark:bg-transparent",
      className,
    )}
    {...props}
  />
);
