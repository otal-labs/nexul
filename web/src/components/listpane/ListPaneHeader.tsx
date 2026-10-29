import type { ReactNode } from "react";
import { PlusIcon } from "lucide-react";

import type { PermissionAction } from "@nexul/client-core/permissionActions";

import { ActionButton } from "@/components/action/ActionButton";
import { Input } from "@/components/ui/input";

interface ListPaneHeaderProps {
  title: string;
  /** Hidden at zero when `hideZeroCount` is set, as the inbox's unread count is. */
  count: number;
  hideZeroCount?: boolean;
  /** The New button's fields, left out on a list nobody creates in (the inbox). */
  newLabel?: string;
  /** The permission creating needs: the button is hidden, or locked with the reason, for a viewer without it. */
  newPermission?: PermissionAction;
  newVerb?: string;
  /** Left out for a list scoped to the workspace, such as plays. */
  projectId?: string;
  onNew?: () => void;
  /** The search band, left out on a list read top down (the inbox). */
  search?: string;
  onSearch?: (value: string) => void;
  /** Extra header controls (a sort toggle), placed left of the new button. */
  controls?: ReactNode;
}

export const ListPaneHeader = ({ title, count, hideZeroCount = false, newLabel, newPermission, newVerb, projectId, onNew, search, onSearch, controls }: ListPaneHeaderProps) => (
  <div className="shrink-0">
    <div className="flex h-14 items-center justify-between border-b border-border pr-2 pl-3">
      <h1 className="flex items-baseline gap-2 text-sm font-semibold">
        {title}
        {(count > 0 || !hideZeroCount) && <span className="font-mono text-xs font-normal text-muted-foreground tabular-nums">{count}</span>}
      </h1>
      <div className="flex items-center gap-1.5">
        {controls}
        {onNew && newPermission && newVerb && newLabel && (
        <ActionButton
          permission={newPermission}
          verb={newVerb}
          {...(projectId && { projectId })}
          variant="ghost"
          size="icon"
          className="size-8"
          aria-label={newLabel}
          title={newLabel}
          icon={PlusIcon}
          onClick={onNew}
        />
        )}
      </div>
    </div>
    {onSearch && (
    <div className="px-3 pt-3 pb-2">
      <Input
        aria-label={`Search ${title.toLowerCase()}`}
        placeholder="Search…"
        value={search ?? ""}
        onChange={(e) => onSearch(e.target.value)}
        className="h-8 text-xs"
      />
    </div>
    )}
  </div>
);
