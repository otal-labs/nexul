import { useState } from "react";

import { Button } from "@/components/ui/button";
import { useInvitationTokenStore } from "@/stores/invitationTokenStore";

interface InvitationNotYouProps {
  // Back to the provider list with the token this tab kept; absent, the person is told to open the link again.
  onRestart: () => void;
}

// The way out of signing in as the wrong account, before anything is written.
export const InvitationNotYou = ({ onRestart }: InvitationNotYouProps) => {
  const kept = useInvitationTokenStore((s) => s.token);
  const [asked, setAsked] = useState(false);
  return (
    <>
      {!asked && <Button type="button" variant="ghost" size="sm" className="-my-1 -mr-3 h-8 shrink-0 text-muted-foreground" onClick={() => (kept ? onRestart() : setAsked(true))}>
        Not you?
      </Button>}
      {asked && <p role="status" className="basis-full pl-[34px] text-xs text-pretty text-muted-foreground">Open the invitation link again and pick another account.</p>}
    </>
  );
};
