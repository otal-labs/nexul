import { Link } from "react-router";

import { SignedOutHeader } from "@/components/auth/SignedOutHeader";
import { InvitationIdentityRow } from "@/components/invitation/InvitationIdentityRow";
import { InvitationNotYou } from "@/components/invitation/InvitationNotYou";
import { Button } from "@/components/ui/button";
import type { RefusedPerson } from "@/models/Invitation";

// Says only that the link is gone: no inviter, nothing that tells expired from deleted, and the way to sign in.
export const InvitationGoneScreen = () => (
  <>
    <SignedOutHeader title="This invitation has been deleted or has expired" subtitle="Ask whoever sent it for a new link." />
    <Button asChild size="lg" variant="outline" className="mt-6 w-full">
      <Link to="/login" replace>Go to sign in</Link>
    </Button>
  </>
);

interface InvitationExistingAccountScreenProps {
  person?: RefusedPerson | undefined;
  // Signs out when this browser holds a session, else goes back to the providers; both keep the same link.
  canRestart: boolean;
  onRestart: () => void;
}

// Turned away before anything is written, so the link stays unused for the person it was made for.
export const InvitationExistingAccountScreen = ({ person, canRestart, onRestart }: InvitationExistingAccountScreenProps) => (
  <>
    <SignedOutHeader title="You already have an account here" subtitle="This invitation is for someone new." />
    {person && <InvitationIdentityRow person={person} className="mt-6" action={<InvitationNotYou canRestart={canRestart} onRestart={onRestart} />} />}
    <Button asChild size="lg" className="mt-4 w-full">
      <Link to="/" replace>Go to {window.location.host}</Link>
    </Button>
    <p className="mt-4 text-xs text-pretty text-muted-foreground">The link stays unused for the person it was meant for.</p>
  </>
);
