import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface MicroheaderProps {
  children: ReactNode;
  id?: string;
  className?: string;
}

// The mono label a group of permission rows sits under.
export const Microheader = ({ children, id, className }: MicroheaderProps) => (
  <h3 id={id} className={cn("font-mono text-[11px] font-normal tracking-wide text-muted-foreground uppercase", className)}>{children}</h3>
);
