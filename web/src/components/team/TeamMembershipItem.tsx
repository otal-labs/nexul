import { useState } from "react";

import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { TeamOverridesForm } from "@/components/team/TeamOverridesForm";
import { TeamReadOnlyReason } from "@/components/team/TeamReadOnlyReason";
import { TeamRoleSelect } from "@/components/team/TeamRoleSelect";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useChangeTeamMemberRole, useRemoveTeamMember } from "@/hooks/TeamHooks";
import type { TeamMembership, TeamPerson, TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamMembershipItemProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
  membership: TeamMembership;
}

const roleChipClass =
  "inline-flex shrink-0 items-center rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] font-medium uppercase tracking-wide text-muted-foreground";

export const TeamMembershipItem = ({ person, workspace, membership }: TeamMembershipItemProps) => {
  const changeRole = useChangeTeamMemberRole();
  const remove = useRemoveTeamMember();
  const { open: confirm } = useConfirmationDialog();
  const [editingOverrides, setEditingOverrides] = useState(false);
  const editable = workspace.can_manage_members && !membership.is_owner;
  const readOnly = !workspace.can_manage_members && !membership.is_owner;
  const hasOverrides = membership.allow.length > 0 || membership.deny.length > 0;
  const target = { workspaceId: workspace.id, userId: person.id };

  const removeFromWorkspace = async () => {
    const ok = await confirm({
      title: `Remove from ${workspace.name}?`,
      message: `${personName(person)} loses access to ${workspace.name}. Their account and what they wrote stay, and you can add them back here.`,
      confirmLabel: "Remove",
    });
    if (ok) remove.mutate(target);
  };

  return (
    <li aria-label={workspace.name} className="space-y-2 px-3 py-2.5">
      <div className="flex items-center gap-2">
        <span className="min-w-0 flex-1 truncate font-medium">{workspace.name}</span>
        {membership.is_owner && <span className={roleChipClass}>Owner</span>}
        {readOnly && <span className="text-sm text-muted-foreground">{membership.role_name}</span>}
        {editable && (
          <TeamRoleSelect
            label={`Role in ${workspace.name}`}
            roles={workspace.roles}
            value={membership.role_id}
            disabled={changeRole.isPending}
            onChange={(roleId) => changeRole.mutate({ ...target, roleId })}
          />
        )}
        {editable && (
          <RowActionsMenu
            subject={workspace.name}
            actions={[
              { label: "Edit overrides", onSelect: () => setEditingOverrides(true) },
              { label: "Remove from workspace", destructive: true, disabled: remove.isPending, onSelect: () => void removeFromWorkspace() },
            ]}
          />
        )}
      </div>
      {membership.is_owner && <p className="text-xs text-muted-foreground">The Owner role can&apos;t be changed or removed, and it bypasses overrides.</p>}
      {readOnly && <TeamReadOnlyReason workspaceName={workspace.name} />}
      {hasOverrides && (
        <p className="text-xs text-muted-foreground">
          Overrides: {membership.allow.length} allowed, {membership.deny.length} denied on top of the role
        </p>
      )}
      {editingOverrides && (
        <TeamOverridesForm key={`${membership.allow.join()}|${membership.deny.join()}`} target={target} membership={membership} onDone={() => setEditingOverrides(false)} />
      )}
    </li>
  );
};
