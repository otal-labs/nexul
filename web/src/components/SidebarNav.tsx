import type { ComponentType } from "react";

import { cn } from "@/lib/utils";

export interface SidebarNavEntry {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  end?: boolean;
  wip?: boolean;
}

export const sectionLabelClass = "px-2.5 pt-4 pb-1 text-[11px] font-medium tracking-wide text-muted-foreground/70";

export const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13.5px] transition-colors duration-150 ease-standard",
    isActive
      ? "bg-accent font-medium text-primary"
      : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
  );
