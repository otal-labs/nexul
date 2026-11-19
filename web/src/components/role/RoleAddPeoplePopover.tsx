import { useState } from "react";
import { UserPlus } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { microheaderClass } from "@/components/Microheader";
import { RoleFindField } from "@/components/role/RoleFindField";
import { RolePersonOption } from "@/components/role/RolePersonOption";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useOpenRole } from "@/hooks/RoleHooks";
import { useFetchTeam, useRoleHolders } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import { AccountStatus, heldRoles } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

// Everyone the viewer's Team shows who doesn't hold the open role: a member gets the role, anyone else joins the
// workspace with it. The list stays open, so several people can be added in a row.
export const RoleAddPeoplePopover = () => {
  const [query, setQuery] = useState("");
  const workspace = useSelectedWorkspace();
  const { role } = useOpenRole();
  const { data: team } = useFetchTeam();
  const holders = useRoleHolders(workspace?.id ?? "", workspace?.name ?? "this workspace");
  if (!role || !workspace) return null;

  const holds = (roleIds: string[]) => roleIds.includes(role.id);
  const candidates = (team?.people ?? []).filter((person) => {
    if (person.status !== AccountStatus.Active) return false;
    const membership = person.workspaces.find((candidate) => candidate.workspace_id === workspace.id);
    return !membership || (!membership.is_owner && !holds(heldRoles(membership).map((held) => held.id)));
  });
  const needle = query.trim().toLowerCase();
  const shown = candidates
    .filter((person) => !needle || `${personName(person)} ${person.login}`.toLowerCase().includes(needle))
    .sort((a, b) => personName(a).localeCompare(personName(b), undefined, { numeric: true }));
  const inWorkspace = shown.filter((person) => person.workspaces.some((m) => m.workspace_id === workspace.id));
  const elsewhere = shown.filter((person) => !inWorkspace.includes(person));

  const pick = (id: string) => {
    const person = candidates.find((candidate) => candidate.id === id);
    if (person) holders.give(person, role);
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <ActionButton permission="members:write" verb="give roles" variant="outline" size="sm" icon={UserPlus} type="button">
          Add people
        </ActionButton>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-88 p-2">
        <div className="flex flex-col gap-1.5">
          <RoleFindField value={query} onChange={setQuery} placeholder="Search people" label="Find a person" />
          <div className="scroll-edge flex max-h-72 flex-col gap-0.5 overflow-y-auto pb-2">
            {inWorkspace.length > 0 && <p className={cn(microheaderClass, "px-2 pt-1.5 pb-0.5")}>In {workspace.name}</p>}
            {inWorkspace.map((person) => (
              <RolePersonOption key={person.id} person={person} disabled={holders.pending} onPick={pick} />
            ))}
            {elsewhere.length > 0 && <p className={cn(microheaderClass, "px-2 pt-1.5 pb-0.5")}>Not in {workspace.name} yet</p>}
            {elsewhere.map((person) => (
              <RolePersonOption key={person.id} person={person} disabled={holders.pending} onPick={pick} />
            ))}
            {team && shown.length === 0 && <p className="px-2 py-3 text-sm text-muted-foreground">{needle ? "Nobody matches." : `Everyone you can see holds ${role.name}.`}</p>}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
