import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { InvitationRow } from "@/components/member/InvitationRow";
import { useFetchInvitations, useRevokeInvitation } from "@/hooks/InvitationHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";

export const InvitationsFeed = () => {
  const { data, isPending, error } = useFetchInvitations();
  const invitations = data?.invitations;
  const inviterName = usePersonNamer();
  const revoke = useRevokeInvitation();

  return (
    <section className="space-y-3" aria-labelledby="active-invitations-title">
      <div className="space-y-1">
        <Microheader id="active-invitations-title">Active invitation links</Microheader>
        <p className="text-sm text-muted-foreground">Each link works once and is shown only when it's created.</p>
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {invitations && invitations.length === 0 && <EmptyRow>No active invitation links</EmptyRow>}
      {invitations && invitations.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {invitations.map((invitation) => (
            <InvitationRow
              key={invitation.id}
              invitation={invitation}
              inviter={inviterName(invitation.invited_by)}
              onRevoke={(id) => revoke.mutate(id)}
              disabled={revoke.isPending}
            />
          ))}
        </EnterList>
      )}
    </section>
  );
};
