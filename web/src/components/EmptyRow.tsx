import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface EmptyRowProps {
  children: ReactNode;
  className?: string;
}

// One quiet row where the list would be; for a whole empty page use EmptyState instead.
export const EmptyRow = ({ children, className }: EmptyRowProps) => (
  <p
    role="status"
    className={cn("rounded-lg border border-border px-4 py-6 text-center text-sm text-muted-foreground", className)}
  >
    {children}
  </p>
);
