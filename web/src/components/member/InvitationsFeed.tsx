import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { InvitationRow } from "@/components/member/InvitationRow";
import { useFetchInvitations, useRevokeInvitation } from "@/hooks/InvitationHooks";

export const InvitationsFeed = () => {
  const { data: invitations, isPending, error } = useFetchInvitations();
  const revoke = useRevokeInvitation();

  return (
    <section className="mt-8 space-y-3" aria-labelledby="active-invitations-title">
      <div><h2 id="active-invitations-title" className="text-sm font-semibold">Active invitation links</h2><p className="text-sm text-muted-foreground">Links are single-use and never shown again after creation.</p></div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {invitations && invitations.length === 0 && <NoDataDisplay size="compact" message="No active invitation links" />}
      {invitations && invitations.length > 0 && <ul className="divide-y divide-border overflow-hidden rounded-md border bg-card shadow-card">{invitations.map((invitation) => <InvitationRow key={invitation.id} invitation={invitation} onRevoke={(id) => revoke.mutate(id)} disabled={revoke.isPending} />)}</ul>}
    </section>
  );
};
