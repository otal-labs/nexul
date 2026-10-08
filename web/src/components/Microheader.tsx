import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

// The one small uppercase label: facts, group labels, rail sections, column heads.
export const microheaderClass = "font-mono text-[11px] font-medium tracking-[0.12em] text-muted-foreground uppercase";

interface MicroheaderProps {
  children: ReactNode;
  id?: string;
  className?: string;
}

export const Microheader = ({ children, id, className }: MicroheaderProps) => (
  <h3 id={id} className={cn(microheaderClass, className)}>
    {children}
  </h3>
);
