import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface LevelRowProps {
  name: string;
  // A mono line under the name, for a summary of what the row holds.
  meta?: string | undefined;
  strong?: boolean;
  className?: string;
  children: ReactNode;
}

// A 44px hairline row straight on its surface: the name on the left, its control trailing right.
export const LevelRow = ({ name, meta, strong = false, className, children }: LevelRowProps) => (
  <li className={cn("flex min-h-11 items-center gap-3 py-1.5", className)}>
    <div className="min-w-0 flex-1">
      <p className={cn("truncate text-sm", strong && "font-medium")}>{name}</p>
      {meta && <p className="truncate font-mono text-[11px] text-muted-foreground">{meta}</p>}
    </div>
    {children}
  </li>
);
