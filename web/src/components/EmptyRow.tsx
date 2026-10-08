import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface EmptyRowProps {
  children: ReactNode;
  className?: string;
}

// One quiet sentence where the list would be; for a whole empty page use EmptyState instead.
export const EmptyRow = ({ children, className }: EmptyRowProps) => (
  <p
    role="status"
    className={cn("px-4 py-5 text-sm text-muted-foreground", className)}
  >
    {children}
  </p>
);
