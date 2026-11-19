import { useState } from "react";
import { ShieldPlus } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { menuItemClass } from "@/components/MenuItem";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { RoleFindField } from "@/components/role/RoleFindField";
import { RoleReach } from "@/components/role/RoleReach";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useRoleChoices } from "@/hooks/RoleHooks";
import { useRoleHolders } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface RoleGivePopoverProps {
  person: TeamPerson;
}

// On No roles: the roles the viewer may give, each with its reach, one pick away from the person who holds none. It never runs past the window:
// it holds to the room it has, scrolls with a fade, and a find field narrows it.
export const RoleGivePopover = ({ person }: RoleGivePopoverProps) => {
  const [query, setQuery] = useState("");
  const workspace = useSelectedWorkspace();
  const { choices } = useRoleChoices(workspace?.id ?? "");
  const holders = useRoleHolders(workspace?.id ?? "", workspace?.name ?? "this workspace");
  const { data: catalog } = useFetchPermissionCatalog();
  const needle = query.trim().toLowerCase();
  const givable = (choices ?? []).filter((choice) => choice.givable);
  const shown = givable.filter((choice) => !needle || `${choice.role.name} ${choice.role.description}`.toLowerCase().includes(needle));

  return (
    <Popover>
      <PopoverTrigger asChild>
        <ActionButton permission="members:write" verb="give roles" variant="outline" size="sm" icon={ShieldPlus} type="button" aria-label={`Give ${personName(person)} a role`}>
          Give a role
        </ActionButton>
      </PopoverTrigger>
      <PopoverContent align="end" className="flex max-h-[min(24rem,var(--radix-popover-content-available-height))] w-96 flex-col gap-1.5 p-2">
        <RoleFindField value={query} onChange={setQuery} placeholder="Search roles" label="Find a role" />
        <div className="scroll-edge flex min-h-0 flex-col gap-1 overflow-y-auto pb-2">
          {shown.map((choice) => (
            <button key={choice.role.id} type="button" className={cn(menuItemClass, "h-auto shrink-0 flex-col items-stretch gap-0 py-2.5")} disabled={holders.pending} onClick={() => holders.give(person, choice.role)}>
              <span className="text-sm text-balance break-words">{choice.role.name}</span>
              {choice.role.description && <span className="block truncate text-xs text-muted-foreground" title={choice.role.description}>{choice.role.description}</span>}
              <span className="mt-1">
                <RoleReach catalog={catalog ?? []} permissions={choice.role.permissions} />
              </span>
            </button>
          ))}
          {choices && shown.length === 0 && <p className="px-2 py-3 text-sm text-muted-foreground">{needle ? "No role matches." : "No role here is yours to give."}</p>}
        </div>
      </PopoverContent>
    </Popover>
  );
};
