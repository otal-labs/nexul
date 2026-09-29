import { UserX } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
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

const overridesLabel = ({ allow, deny }: TeamMembership): string => {
  if (allow.length === 0 && deny.length === 0) return "No overrides on top of the role";
  return `Overrides: ${allow.length} allowed, ${deny.length} denied on top of the role`;
};

const roleChipClass =
  "inline-flex shrink-0 items-center rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] font-medium uppercase tracking-wide text-muted-foreground";

export const TeamMembershipItem = ({ person, workspace, membership }: TeamMembershipItemProps) => {
  const changeRole = useChangeTeamMemberRole();
  const remove = useRemoveTeamMember();
  const { open: confirm } = useConfirmationDialog();
  const editable = workspace.can_manage_members && !membership.is_owner;
  const readOnly = !workspace.can_manage_members && !membership.is_owner;
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
    <>
      <div className="flex flex-wrap items-center gap-2">
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
          <Button type="button" variant="ghost" size="icon" aria-label={`Remove from ${workspace.name}`} title="Remove from workspace" disabled={remove.isPending} onClick={() => void removeFromWorkspace()}>
            <UserX className="size-4" />
          </Button>
        )}
      </div>
      {membership.is_owner && <p className="text-xs text-muted-foreground">The Owner role can&apos;t be changed or removed, and it bypasses overrides.</p>}
      {readOnly && <TeamReadOnlyReason workspaceName={workspace.name} />}
      {readOnly && <p className="text-xs text-muted-foreground">{overridesLabel(membership)}</p>}
      {editable && (
        <Collapsible>
          <CollapsibleTrigger type="button" className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
            {overridesLabel(membership)} · Edit
          </CollapsibleTrigger>
          <CollapsibleContent className="mt-3">
            <TeamOverridesForm key={`${membership.allow.join()}|${membership.deny.join()}`} target={target} membership={membership} />
          </CollapsibleContent>
        </Collapsible>
      )}
    </>
  );
};
