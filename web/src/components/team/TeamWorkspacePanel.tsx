import { UserMinus } from "lucide-react";

import { SettingsHint } from "@/components/settings/SettingsHint";
import { Button } from "@/components/ui/button";
import { Microheader } from "@/components/Microheader";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { RolesChecklist } from "@/components/role/RolesChecklist";
import { TeamReadOnlyReason } from "@/components/team/TeamReadOnlyReason";
import { TeamRemovedNotice } from "@/components/team/TeamRemovedNotice";
import { TeamViewAsButton } from "@/components/team/TeamViewAsButton";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import { heldRoles, type TeamMembership, type TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamWorkspacePanelProps {
  workspace: TeamWorkspace;
  // As it will read after Confirm.
  membership: TeamMembership;
}

export const TeamWorkspacePanel = ({ workspace, membership }: TeamWorkspacePanelProps) => {
  const { person, draft, dispatch } = useMemberDraft();
  const entry = draft[workspace.id];
  const editable = workspace.can_manage_members && !membership.is_owner;
  const removed = entry?.removed === true;
  const firstName = personName(person).split(" ")[0] ?? "";
  const roleIds = heldRoles(membership).map((role) => role.id);
  const emptyNote = entry?.added
    ? `Pick at least one role to add ${firstName} to ${workspace.name}.`
    : `With no roles, ${firstName} sees no projects or plays in ${workspace.name}.`;

  return (
    <div className="space-y-5">
      {removed && <TeamRemovedNotice workspace={workspace} />}
      {!removed && editable && (
        <RolesChecklist
          workspaceId={workspace.id}
          value={roleIds}
          onChange={(next) => dispatch({ type: "roles", workspaceId: workspace.id, roleIds: next })}
          title="Roles"
          emptyNote={emptyNote}
        />
      )}
      {!removed && !editable && (
        <section className="space-y-2">
          <Microheader>Roles</Microheader>
          <RoleChipRow roles={heldRoles(membership)} />
          {membership.is_owner && <SettingsHint>The Owner role is never given or taken here, and it holds everything in {workspace.name}.</SettingsHint>}
          {!membership.is_owner && <TeamReadOnlyReason workspaceName={workspace.name} />}
        </section>
      )}
      {!removed && !membership.is_owner && (
        <SettingsHint>
          The projects, channels and plays {firstName} sees follow from these roles and from the rules on each one.
        </SettingsHint>
      )}
      {!removed && !entry && !membership.is_owner && <TeamViewAsButton person={person} workspace={workspace} />}
      {!removed && editable && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2 text-muted-foreground hover:text-destructive"
          onClick={() => dispatch({ type: "remove", workspaceId: workspace.id, removed: true })}
        >
          <UserMinus className="size-4" />
          Remove from workspace
        </Button>
      )}
    </div>
  );
};
