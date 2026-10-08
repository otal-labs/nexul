import type { ComponentType } from "react";

import { microheaderClass } from "@/components/Microheader";
import type { RouteArea } from "@/models/Access";
import { cn } from "@/lib/utils";

export interface SidebarNavEntry {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  area: RouteArea;
  end?: boolean;
  wip?: boolean;
}

export const sectionLabelClass = cn(microheaderClass, "px-2.5 pt-4 pb-1");

export const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors duration-150 ease-standard",
    isActive
      ? "font-medium text-accent-foreground [&_svg]:text-brand"
      : "text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground",
  );
