import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsHintProps {
  children: ReactNode;
  className?: string;
}

// The third level of a settings card's type: a description or hint under a card title or a row label. A block span, so it nests in a label or a paragraph too.
export const SettingsHint = ({ children, className }: SettingsHintProps) => (
  <span className={cn("block text-xs leading-normal text-pretty text-muted-foreground", className)}>{children}</span>
);
