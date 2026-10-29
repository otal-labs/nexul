import { InvitationIdentityRow } from "@/components/invitation/InvitationIdentityRow";
import { InvitationNotYou } from "@/components/invitation/InvitationNotYou";
import { Button } from "@/components/ui/button";
import type { RefusedPerson } from "@/models/Invitation";

interface InvitationAcceptancePanelProps {
  person?: RefusedPerson | undefined;
  pending: boolean;
  onAccept: () => void;
  onDecline: () => void;
  onRestart: () => void;
}

// Decline leaves the link unused, the same as walking away.
export const InvitationAcceptancePanel = ({ person, pending, onAccept, onDecline, onRestart }: InvitationAcceptancePanelProps) => (
  <div className="mt-5 space-y-3">
    {person && <InvitationIdentityRow person={person} action={<InvitationNotYou onRestart={onRestart} />} />}
    <div className="grid gap-1.5">
      <Button type="button" size="lg" className="w-full" loading={pending} onClick={onAccept}>Accept invitation</Button>
      <Button type="button" variant="ghost" className="w-full text-muted-foreground" onClick={onDecline}>Decline</Button>
    </div>
  </div>
);
