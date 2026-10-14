import { Plus } from "lucide-react";

import type { PermissionAction } from "@nexul/client-core/permissionActions";

import { ActionButton } from "@/components/action/ActionButton";
import { sectionLabelClass } from "@/components/SidebarNav";
import { cn } from "@/lib/utils";

interface SidebarSectionHeaderProps {
  label: string;
  // Names the + for screen readers and its tooltip, e.g. "New channel".
  actionLabel: string;
  // The permission creating here needs; the + is hidden, or locked with the reason, for a viewer without it.
  permission: PermissionAction;
  verb: string;
  onAction: () => void;
}

// A sidebar section label with its create action; the + is never hover-only, since that hides the way in.
export const SidebarSectionHeader = ({ label, actionLabel, permission, verb, onAction }: SidebarSectionHeaderProps) => (
  <div className={cn(sectionLabelClass, "flex items-center justify-between pr-1")}>
    <span>{label}</span>
    <ActionButton
      permission={permission}
      verb={verb}
      icon={Plus}
      variant="ghost"
      size="icon"
      onClick={onAction}
      aria-label={actionLabel}
      title={actionLabel}
      className="size-5 rounded-md text-muted-foreground/70 hover:bg-accent hover:text-foreground [&_svg]:size-3.5"
    />
  </div>
);
