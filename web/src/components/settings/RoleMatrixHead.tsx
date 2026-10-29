import { CrownIcon } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useRoleActions } from "@/hooks/useRoleActions";
import { headName, type Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

interface RoleMatrixHeadProps {
  role: Role;
  workspaceId: string;
  holders: TeamPerson[] | undefined;
}

// A role's column head: its name, how many hold it (named in its tooltip), and its menu (the Owner's has none).
export const RoleMatrixHead = ({ role, workspaceId, holders }: RoleMatrixHeadProps) => {
  const { actions, dialogs } = useRoleActions(role, workspaceId, holders);
  return (
    <th scope="col" className="sticky top-0 z-20 w-24 min-w-24 snap-start border-b border-border bg-card px-2 pb-1.5 text-left align-bottom font-normal">
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="line-clamp-3 text-xs leading-tight font-medium text-balance break-words text-foreground">
            {role.is_owner_role && <CrownIcon className="mr-1 inline size-3 -translate-y-px text-muted-foreground" aria-hidden />}
            {headName(role.name, 13)}
          </span>
        </TooltipTrigger>
        <TooltipContent>{role.name}</TooltipContent>
      </Tooltip>
      <span className="mt-0.5 flex h-6 items-center justify-between font-mono text-[11px] text-muted-foreground tabular-nums">
        {holders && (
          <span className="translate-y-px" aria-label={holders.length === 1 ? "1 person" : `${holders.length} people`} title={holders.map((p) => p.display_name || p.login).join(", ")}>
            {holders.length === 0 ? "nobody" : holders.length}
          </span>
        )}
        {!role.is_owner_role && <RowActionsMenu subject={role.name} actions={actions} />}
      </span>
      {dialogs}
    </th>
  );
};
