import { useState } from "react";

import { Button } from "@/components/ui/button";

interface InvitationNotYouProps {
  // False only when the page holds no link or acceptance to start another sign-in from; then it says to open the link again.
  canRestart: boolean;
  onRestart: () => void;
}

// The way out of signing in as the wrong account, before anything is written.
export const InvitationNotYou = ({ canRestart, onRestart }: InvitationNotYouProps) => {
  const [asked, setAsked] = useState(false);
  return (
    <>
      {!asked && (
        <Button type="button" variant="ghost" size="sm" className="-my-1 -mr-3 h-8 shrink-0 text-[13px] text-muted-foreground" onClick={() => (canRestart ? onRestart() : setAsked(true))}>
          Not you?
        </Button>
      )}
      {asked && <p role="status" className="basis-full pl-[34px] text-xs text-pretty text-muted-foreground">Open the invitation link again and pick another account.</p>}
    </>
  );
};
