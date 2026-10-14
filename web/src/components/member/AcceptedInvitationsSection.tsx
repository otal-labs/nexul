import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { AcceptedInvitationRow } from "@/components/member/AcceptedInvitationRow";
import { useFetchInvitations } from "@/hooks/InvitationHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";

// The links accepted in the last 30 days, newest first, each with the person it let in; the list above shows its
// loading and errors.
export const AcceptedInvitationsSection = ({ onOpen }: { onOpen: (id: string) => void }) => {
  const { data } = useFetchInvitations();
  const { data: team } = useFetchTeam();
  const inviterName = usePersonNamer();
  const rows = (data?.accepted ?? []).flatMap((invitation) => {
    const person = team?.people.find((candidate) => candidate.id === invitation.redeemed_by);
    return person ? [{ invitation, person }] : [];
  });
  return (
    <section className="space-y-3" aria-labelledby="accepted-invitations-title">
      <div className="space-y-1">
        <Microheader id="accepted-invitations-title">Accepted in the last 30 days</Microheader>
        <p className="text-sm text-muted-foreground">Who each link let in, and how far their setup got.</p>
      </div>
      {data && team && rows.length === 0 && <EmptyRow>No one accepted a link in the last 30 days</EmptyRow>}
      {rows.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {rows.map(({ invitation, person }) => (
            <AcceptedInvitationRow key={invitation.id} invitation={invitation} person={person} inviter={inviterName(invitation.invited_by)} onOpen={onOpen} />
          ))}
        </EnterList>
      )}
    </section>
  );
};
