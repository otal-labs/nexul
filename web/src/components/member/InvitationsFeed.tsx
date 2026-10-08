import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { InvitationRow } from "@/components/member/InvitationRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchInvitations, useRevokeInvitation } from "@/hooks/InvitationHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { personName } from "@/utils/TeamUtility";

export const InvitationsFeed = () => {
  const { data: invitations, isPending, error } = useFetchInvitations();
  const { data: team } = useFetchTeam();
  const { data: me } = useFetchMe();
  const revoke = useRevokeInvitation();

  // The Team the section already holds names the inviter; someone outside the viewer's view stays unnamed.
  const inviterName = (id: string): string => {
    const person = team?.people.find((candidate) => candidate.id === id);
    if (person) return personName(person);
    if (id === me?.user.id) return "you";
    return "another member";
  };

  return (
    <section className="mt-8 space-y-3" aria-labelledby="active-invitations-title">
      <div className="space-y-1">
        <Microheader id="active-invitations-title">Active invitation links</Microheader>
        <p className="text-sm text-muted-foreground">Links are single-use and never shown again after creation.</p>
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
