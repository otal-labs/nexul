import { InvitationIdentityRow } from "@/components/invitation/InvitationIdentityRow";
import { Button } from "@/components/ui/button";
import type { RefusedPerson } from "@/models/Invitation";

interface InvitationAcceptancePanelProps {
  person?: RefusedPerson | undefined;
  pending: boolean;
  onAccept: () => void;
  onDecline: () => void;
}

// Decline leaves the link unused, the same as walking away.
export const InvitationAcceptancePanel = ({ person, pending, onAccept, onDecline }: InvitationAcceptancePanelProps) => (
  <div className="mt-3">
    <p className="text-xs text-muted-foreground">You may be added to more projects later.</p>
    <div className="mt-7 space-y-3">
      {person && <InvitationIdentityRow person={person} />}
      <div className="grid grid-cols-2 gap-2">
        <Button type="button" variant="outline" onClick={onDecline}>Decline</Button>
        <Button type="button" loading={pending} onClick={onAccept}>Accept invitation</Button>
      </div>
    </div>
  </div>
);
