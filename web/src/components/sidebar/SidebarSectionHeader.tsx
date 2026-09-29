import { Plus } from "lucide-react";

import { sectionLabelClass } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";

interface SidebarSectionHeaderProps {
  label: string;
  // Names the + for screen readers and its tooltip, e.g. "New channel".
  actionLabel: string;
  // Absent for a viewer who may not create here: the + is hidden, never shown disabled.
  onAction?: (() => void) | undefined;
}

// A sidebar section label with its create action; the + is never hover-only, since that hides the way in.
export const SidebarSectionHeader = ({ label, actionLabel, onAction }: SidebarSectionHeaderProps) => (
  <div className={cn(sectionLabelClass, "flex items-center justify-between pr-1")}>
    <span>{label}</span>
    {onAction && (
      <button
        type="button"
        onClick={onAction}
        aria-label={actionLabel}
        title={actionLabel}
        className="flex size-5 items-center justify-center rounded-md text-muted-foreground/70 transition-colors hover:bg-accent hover:text-foreground"
      >
        <Plus className="size-3.5" aria-hidden />
      </button>
    )}
  </div>
);
