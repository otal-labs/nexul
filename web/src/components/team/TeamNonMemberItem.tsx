import { useState } from "react";
import { UserPlus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { TeamReadOnlyReason } from "@/components/team/TeamReadOnlyReason";
import { TeamRoleSelect } from "@/components/team/TeamRoleSelect";
import { useAddTeamMember } from "@/hooks/TeamHooks";
import type { TeamPerson, TeamWorkspace } from "@/models/Team";

interface TeamNonMemberItemProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
}

export const TeamNonMemberItem = ({ person, workspace }: TeamNonMemberItemProps) => {
  const add = useAddTeamMember();
  const assignable = workspace.roles.filter((role) => !role.is_owner);
  const [picked, setPicked] = useState("");
  const roleId = picked || assignable[0]?.id || "";
  const removed = person.status === "removed";
  const canAdd = workspace.can_manage_members && !removed;

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <span className="min-w-0 flex-1 truncate font-medium text-muted-foreground">{workspace.name}</span>
        <span className="text-xs text-muted-foreground">Not a member</span>
        {canAdd && assignable.length > 0 && (
          <TeamRoleSelect label={`Role to add in ${workspace.name}`} roles={workspace.roles} value={roleId} onChange={setPicked} />
        )}
        {canAdd && assignable.length > 0 && (
          <Button type="button" variant="outline" size="sm" disabled={add.isPending} onClick={() => add.mutate({ workspaceId: workspace.id, userId: person.id, roleId })}>
            <UserPlus className="size-4" />Add
          </Button>
        )}
      </div>
      {canAdd && assignable.length === 0 && <p className="text-xs text-muted-foreground">{workspace.name} has no role to give yet; create one under Roles in that workspace.</p>}
      {removed && <p className="text-xs text-muted-foreground">Restore the account to give it workspace access.</p>}
      {!removed && !workspace.can_manage_members && <TeamReadOnlyReason workspaceName={workspace.name} />}
    </>
  );
};
