import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { RoleInvitationRow } from "@/components/role/RoleInvitationRow";
import { useFetchInvitations, useRevokeInvitation } from "@/hooks/InvitationHooks";
import { useOpenRole } from "@/hooks/RoleHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";
import { invitationsGiving } from "@/models/Invitation";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The open links that give the open role here; a role nobody is invited to shows nothing.
export const RoleInvitationsSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { roleId, role } = useOpenRole();
  const { data } = useFetchInvitations();
  const inviterName = usePersonNamer();
  const revoke = useRevokeInvitation();
  const pending = data && roleId ? invitationsGiving(data.invitations, workspaceId, roleId) : [];

  if (pending.length === 0) return null;

  return (
    <section className="space-y-2" aria-labelledby="role-invitations-title">
      <div className="flex items-baseline gap-2">
        <Microheader id="role-invitations-title">Invited</Microheader>
        <span className="font-mono text-micro text-muted-foreground tabular-nums">{pending.length}</span>
      </div>
      <EnterList aria-label="Open invitation links" className="border-t border-border">
        {pending.map((invitation) => (
          <RoleInvitationRow
            key={invitation.id}
            invitation={invitation}
            openName={role?.name ?? ""}
            workspaceId={workspaceId}
            inviter={inviterName(invitation.invited_by)}
            onRevoke={(id) => revoke.mutate(id)}
          />
        ))}
      </EnterList>
    </section>
  );
};
