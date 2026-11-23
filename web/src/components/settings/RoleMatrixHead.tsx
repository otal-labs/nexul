import { CrownIcon } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { OverflowCount } from "@/components/OverflowCount";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useRoleActions } from "@/hooks/useRoleActions";
import { headName, type Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

const holdersLabel = (count: number): string => {
  if (count === 0) return "Nobody";
  if (count === 1) return "1 person";
  return `${count} people`;
};

interface RoleMatrixHeadProps {
  role: Role;
  workspaceId: string;
  holders: TeamPerson[] | undefined;
}

// A role's column head: its name, how many hold it (a count that lists them when opened), and its menu (the Owner's has none).
export const RoleMatrixHead = ({ role, workspaceId, holders }: RoleMatrixHeadProps) => {
  const { actions, dialogs } = useRoleActions(role, workspaceId, holders);
  return (
    <th scope="col" data-role-head="" className="group/head sticky top-0 z-20 min-w-24 snap-start border-b border-border bg-card px-2 pt-3 pb-1.5 text-left align-bottom font-normal">
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="line-clamp-3 text-xs leading-tight font-medium text-balance break-words text-foreground">
            {role.is_owner_role && <CrownIcon className="mr-1 inline size-3 -translate-y-px text-muted-foreground" aria-hidden />}
            {headName(role.name, 13)}
          </span>
        </TooltipTrigger>
        <TooltipContent>{role.name}</TooltipContent>
      </Tooltip>
      <span className="mt-0.5 flex h-6 items-center gap-2 font-mono text-micro text-muted-foreground tabular-nums">
        {holders && holders.length === 0 && (
          <span aria-label={holdersLabel(0)} className="font-sans text-xs">
            &mdash;
          </span>
        )}
        {holders && holders.length > 0 && (
          <OverflowCount label={holdersLabel(holders.length)} items={holders.map((p) => p.display_name || p.login)}>
            {holders.length}
          </OverflowCount>
        )}
        {!role.is_owner_role && (
          <span className="flex [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/head:opacity-100 [@media(hover:hover)]:group-hover/head:opacity-100 [@media(hover:hover)]:has-data-[state=open]:opacity-100">
            <RowActionsMenu subject={role.name} actions={actions} />
          </span>
        )}
      </span>
      {dialogs}
    </th>
  );
};
