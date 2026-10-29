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
  // Set while this browser holds a session: signing out returns the card to the providers with the same link.
  onSignOut?: (() => void) | undefined;
  onRestart: () => void;
}

// Turned away before anything is written, so the link stays unused for the person it was made for.
export const InvitationExistingAccountScreen = ({ person, onSignOut, onRestart }: InvitationExistingAccountScreenProps) => (
  <>
    <SignedOutHeader title="You already have an account here" subtitle="This invitation is for someone new." />
    {person && <InvitationIdentityRow person={person} className="mt-6" action={!onSignOut && <InvitationNotYou onRestart={onRestart} />} />}
    <div className="mt-4 grid gap-1.5">
      <Button asChild size="lg" className="w-full">
        <Link to="/" replace>Go to {window.location.host}</Link>
      </Button>
      {onSignOut && <Button type="button" size="lg" variant="ghost" className="w-full text-muted-foreground" onClick={onSignOut}>Sign out and use another account</Button>}
    </div>
    <p className="mt-4 text-xs text-pretty text-muted-foreground">The link stays unused for the person it was meant for.</p>
  </>
);
