import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface EmptyRowProps {
  children: ReactNode;
  /** Lines up with the text around it (a card body) instead of padding like a row of a list. */
  flush?: boolean;
  className?: string;
}

// One quiet sentence where the list would be; for a whole empty page use EmptyState instead.
export const EmptyRow = ({ children, flush = false, className }: EmptyRowProps) => (
  <p
    role="status"
    className={cn("text-sm text-muted-foreground", !flush && "px-4 py-5", className)}
  >
    {children}
  </p>
);
