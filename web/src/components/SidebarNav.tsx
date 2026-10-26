import type { ComponentType } from "react";

import { microheaderClass } from "@/components/Microheader";
import type { RouteArea } from "@/models/Access";
import { cn } from "@/lib/utils";

export interface SidebarNavEntry {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  // Absent for a page every member opens.
  area?: RouteArea;
  end?: boolean;
  wip?: boolean;
}

export const sectionLabelClass = cn(microheaderClass, "px-2.5 pt-4 pb-1");

export const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "nav-row relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm",
    isActive
      ? "font-medium text-foreground"
      : "text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground active:bg-accent",
  );

// On the icon rail a count sits on its icon's top-right corner, cut out of the icon by a canvas-coloured ring.
export const railBadgeClass = "absolute top-0.5 left-[25px] flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-background px-[3px] text-[10px] ring-2 ring-background";
