import { KeyRound } from "lucide-react";

import { errorMessage } from "@/api/client";
import { InlineNotice } from "@/components/InlineNotice";
import { Button } from "@/components/ui/button";
import { usePasskeySignIn } from "@/hooks/PasskeyHooks";
import { passkeyCancelled } from "@/lib/passkeys";

// Its mark sits where the provider buttons' marks do, so the stack lines up.
export const PasskeySignInButton = () => {
  const signIn = usePasskeySignIn();
  const failed = signIn.error != null && !passkeyCancelled(signIn.error);

  return (
    <>
      <Button
        type="button"
        size="lg"
        variant="outline"
        className="relative w-full [&>svg]:absolute [&>svg]:left-4"
        loading={signIn.isPending}
        onClick={() => signIn.mutate()}
      >
        <KeyRound />
        Sign in with a passkey
      </Button>
      {failed && <InlineNotice title="Couldn't sign you in">{errorMessage(signIn.error)}</InlineNotice>}
    </>
  );
};
