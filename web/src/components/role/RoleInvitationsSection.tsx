import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { InvitationRow } from "@/components/member/InvitationRow";
import { useFetchInvitations, useRevokeInvitation } from "@/hooks/InvitationHooks";
import { useOpenRole } from "@/hooks/RoleHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";
import { invitationsGiving } from "@/models/Invitation";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The open links that give the open role here; a role nobody is invited to shows nothing.
export const RoleInvitationsSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { roleId } = useOpenRole();
  const { data } = useFetchInvitations();
  const inviterName = usePersonNamer();
  const revoke = useRevokeInvitation();
  const pending = data && roleId ? invitationsGiving(data.invitations, workspaceId, roleId) : [];

  if (pending.length === 0) return null;

  return (
    <section className="space-y-2" aria-labelledby="role-invitations-title">
      <Microheader id="role-invitations-title">Invited · {pending.length}</Microheader>
      <EnterList className="divide-y divide-border border-y border-border">
        {pending.map((invitation) => (
          <InvitationRow key={invitation.id} invitation={invitation} inviter={inviterName(invitation.invited_by)} onRevoke={(id) => revoke.mutate(id)} disabled={revoke.isPending} />
        ))}
      </EnterList>
    </section>
  );
};
