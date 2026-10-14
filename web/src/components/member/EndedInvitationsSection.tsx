import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { EndedInvitationRow } from "@/components/member/EndedInvitationRow";
import { useFetchInvitations } from "@/hooks/InvitationHooks";
import { usePersonNamer } from "@/hooks/usePersonNamer";

// The links that expired or were deleted in the last 30 days, newest first; the list above shows loading and errors.
export const EndedInvitationsSection = () => {
  const { data } = useFetchInvitations();
  const inviterName = usePersonNamer();
  return (
    <section className="space-y-3" aria-labelledby="ended-invitations-title">
      <div className="space-y-1">
        <Microheader id="ended-invitations-title">Expired or deleted in the last 30 days</Microheader>
        <p className="text-sm text-muted-foreground">Links that stopped working before anyone used them.</p>
      </div>
      {data && data.ended.length === 0 && <EmptyRow>No link expired or was deleted in the last 30 days</EmptyRow>}
      {data && data.ended.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {data.ended.map((invitation) => (
            <EndedInvitationRow key={invitation.id} invitation={invitation} inviter={inviterName(invitation.invited_by)} />
          ))}
        </EnterList>
      )}
    </section>
  );
};
