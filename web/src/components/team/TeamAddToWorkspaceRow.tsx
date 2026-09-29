import { useState } from "react";
import { UserPlus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { TeamRoleSelect } from "@/components/team/TeamRoleSelect";
import { useAddTeamMember } from "@/hooks/TeamHooks";
import type { TeamPerson, TeamWorkspace } from "@/models/Team";
import { membershipIn } from "@/utils/TeamUtility";

interface TeamAddToWorkspaceRowProps {
  person: TeamPerson;
  workspaces: TeamWorkspace[];
}

// Offers only workspaces the person is not in, the viewer manages, and that have a role to give.
export const TeamAddToWorkspaceRow = ({ person, workspaces }: TeamAddToWorkspaceRowProps) => {
  const add = useAddTeamMember();
  const [pickedWorkspace, setPickedWorkspace] = useState("");
  const [pickedRole, setPickedRole] = useState("");
  const removed = person.status === "removed";
  const candidates = workspaces.filter(
    (workspace) => workspace.can_manage_members && !membershipIn(person, workspace.id) && workspace.roles.some((role) => !role.is_owner),
  );
  const workspace = candidates.find((candidate) => candidate.id === pickedWorkspace) ?? candidates[0];
  const assignable = workspace?.roles.filter((role) => !role.is_owner) ?? [];
  const roleId = assignable.find((role) => role.id === pickedRole)?.id ?? assignable[0]?.id ?? "";

  const pickWorkspace = (id: string) => {
    setPickedWorkspace(id);
    setPickedRole("");
  };

  return (
    <>
      {removed && <p className="text-xs text-muted-foreground">Restore the account to give it workspace access.</p>}
      {!removed && workspace && (
        <div className="flex flex-wrap items-center gap-2 rounded-md border border-dashed border-border px-3 py-2.5">
          <span className="min-w-0 flex-1 text-sm text-muted-foreground">Add to a workspace</span>
          <Select value={workspace.id} onValueChange={pickWorkspace}>
            <SelectTrigger aria-label="Workspace to add to" className="h-8 w-auto min-w-28 shrink-0">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {candidates.map((candidate) => (
                <SelectItem key={candidate.id} value={candidate.id}>
                  {candidate.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <TeamRoleSelect label={`Role to add in ${workspace.name}`} roles={workspace.roles} value={roleId} onChange={setPickedRole} />
          <Button type="button" variant="outline" size="sm" loading={add.isPending} onClick={() => add.mutate({ workspaceId: workspace.id, userId: person.id, roleId })}>
            <UserPlus className="size-4" />Add
          </Button>
        </div>
      )}
    </>
  );
};
