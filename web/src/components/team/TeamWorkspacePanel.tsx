import { useState } from "react";
import { UserMinus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Microheader } from "@/components/Microheader";
import { TeamOverridesForm } from "@/components/team/TeamOverridesForm";
import { TeamProjectAccess } from "@/components/team/TeamProjectAccess";
import { TeamReadOnlyReason } from "@/components/team/TeamReadOnlyReason";
import { TeamRemovedNotice } from "@/components/team/TeamRemovedNotice";
import { TeamRoleSelect } from "@/components/team/TeamRoleSelect";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import type { TeamMembership, TeamWorkspace } from "@/models/Team";

interface TeamWorkspacePanelProps {
  workspace: TeamWorkspace;
  // As it will read after Confirm.
  membership: TeamMembership;
}

const roleChipClass =
  "inline-flex shrink-0 items-center rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] font-medium uppercase tracking-wide text-muted-foreground";

export const TeamWorkspacePanel = ({ workspace, membership }: TeamWorkspacePanelProps) => {
  const { draft, dispatch } = useMemberDraft();
  const [editingOverrides, setEditingOverrides] = useState(false);
  const editable = workspace.can_manage_members && !membership.is_owner;
  const readOnly = !workspace.can_manage_members && !membership.is_owner;
  const hasOverrides = membership.allow.length > 0 || membership.deny.length > 0;
  const removed = draft[workspace.id]?.removed === true;

  return (
    <div className="space-y-5">
      {removed && <TeamRemovedNotice workspace={workspace} />}
      {!removed && (
        <section className="space-y-2">
          <div className="flex min-h-9 items-center gap-2">
            <Microheader className="flex-1">Role</Microheader>
            {membership.is_owner && <span className={roleChipClass}>Owner</span>}
            {readOnly && <span className="text-sm text-muted-foreground">{membership.role_name}</span>}
            {editable && (
              <TeamRoleSelect
                label={`Role in ${workspace.name}`}
                roles={workspace.roles}
                value={membership.role_id}
                onChange={(roleId) => dispatch({ type: "change", workspaceId: workspace.id, change: { roleId } })}
              />
            )}
          </div>
          {membership.is_owner && <p className="text-xs text-muted-foreground">The Owner role can&apos;t be changed or removed, and it bypasses overrides and Project access.</p>}
          {readOnly && <TeamReadOnlyReason workspaceName={workspace.name} />}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            {hasOverrides && <span>Overrides: {membership.allow.length} allowed, {membership.deny.length} denied on top of the role</span>}
            {editable && (
              <button type="button" className="underline-offset-4 hover:text-foreground hover:underline" onClick={() => setEditingOverrides(!editingOverrides)}>
                {editingOverrides ? "Hide overrides" : "Edit overrides"}
              </button>
            )}
          </div>
          {editingOverrides && <TeamOverridesForm workspaceId={workspace.id} membership={membership} />}
        </section>
      )}
      {!removed && !membership.is_owner && <TeamProjectAccess workspace={workspace} membership={membership} />}
      {!removed && editable && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-destructive"
          onClick={() => dispatch({ type: "remove", workspaceId: workspace.id, removed: true })}
        >
          <UserMinus className="size-4" />
          Remove from workspace
        </Button>
      )}
    </div>
  );
};
