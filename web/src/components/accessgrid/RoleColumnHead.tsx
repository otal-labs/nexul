import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { AccessGridRole } from "@/models/AccessGrid";
import { headName } from "@/models/Role";
import { headText } from "@/components/accessgrid/gridStyles";
import { cn } from "@/lib/utils";

interface RoleColumnHeadProps {
  role: AccessGridRole;
}

// The role's name over how many people hold it, so a tick says how many people it lets in.
export const RoleColumnHead = ({ role }: RoleColumnHeadProps) => (
  <th scope="col" className="sticky top-0 z-20 w-18 min-w-18 snap-start border-b border-border bg-card px-0 pb-2 align-top">
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn(headText, "line-clamp-3 w-18 text-center text-balance break-words")}>{headName(role.name)}</span>
      </TooltipTrigger>
      <TooltipContent>{role.name}</TooltipContent>
    </Tooltip>
    <span className="mt-0.5 block text-center font-mono text-micro text-muted-foreground tabular-nums" aria-label={`${role.holders} holding it`}>
      {role.holders === 0 ? "nobody" : role.holders}
    </span>
  </th>
);
