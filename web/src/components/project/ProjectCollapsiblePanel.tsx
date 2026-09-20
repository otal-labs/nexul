import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface ProjectCollapsiblePanelProps {
  open: boolean;
  children: ReactNode;
}

// Tweens the grid track 0fr->1fr with the wrapper clipping overflow, so height animates without JS measurement.
export const ProjectCollapsiblePanel = ({ open, children }: ProjectCollapsiblePanelProps) => (
  <div
    className={cn(
      "grid transition-[grid-template-rows] duration-200 ease-out",
      open ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
    )}
  >
    <div
      className={cn(
        "overflow-hidden transition-opacity duration-200 ease-out",
        open ? "opacity-100" : "opacity-0",
      )}
    >
      {children}
    </div>
  </div>
);
