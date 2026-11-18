import { ShieldPlus } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { menuItemClass } from "@/components/MenuItem";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useRoleChoices } from "@/hooks/RoleHooks";
import { useRoleHolders } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface RoleGivePopoverProps {
  person: TeamPerson;
}

// On No roles: the roles the viewer may give, one pick away from the person who holds none.
export const RoleGivePopover = ({ person }: RoleGivePopoverProps) => {
  const workspace = useSelectedWorkspace();
  const { choices } = useRoleChoices(workspace?.id ?? "");
  const holders = useRoleHolders(workspace?.id ?? "", workspace?.name ?? "this workspace");
  const givable = (choices ?? []).filter((choice) => choice.givable);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <ActionButton permission="members:write" verb="give roles" variant="outline" size="sm" icon={ShieldPlus} type="button" aria-label={`Give ${personName(person)} a role`}>
          Give a role
        </ActionButton>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-1">
        {givable.map((choice) => (
          <button key={choice.role.id} type="button" className={cn(menuItemClass, "h-auto items-start py-1.5")} disabled={holders.pending} onClick={() => holders.give(person, choice.role)}>
            <span className="min-w-0 flex-1">
              <span className="block text-sm text-balance">{choice.role.name}</span>
              {choice.role.description && <span className="block truncate text-xs text-muted-foreground">{choice.role.description}</span>}
            </span>
          </button>
        ))}
        {choices && givable.length === 0 && <p className="px-2 py-3 text-sm text-muted-foreground">No role here is yours to give.</p>}
      </PopoverContent>
    </Popover>
  );
};
