import type { ComponentType } from "react";

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

// Sentence case in the sans, quieter than the rows: a label names the group, it is not a row to read.
export const sectionLabelClass = "px-2.5 pt-5 pb-1 text-xs font-medium text-muted-foreground/80";

export const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "nav-row relative flex items-center gap-2.5 rounded-md px-2.5 py-1 text-[13px]",
    isActive
      ? "font-medium text-foreground"
      : "text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground active:bg-accent",
  );

// On the icon rail every badge is one dot clear of its icon's top-right corner: a canvas ring cutting into the icon showed dark on the current row's tint.
export const railDotClass = "absolute top-0.5 left-[34px] size-2 rounded-full";
