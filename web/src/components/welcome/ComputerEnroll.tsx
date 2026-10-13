import { useEffect, useRef } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AddComputerStep } from "@/components/pairing/AddComputerStep";
import { Button } from "@/components/ui/button";
import { WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { useEnrollComputer } from "@/hooks/ComputerHooks";
import { addedWithRunner, pairedThroughRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface ComputerEnrollProps {
  // The person's computers as the list last read them, which every pushed frame refreshes.
  computers: Computer[];
  onSetUp: (computerId: string) => void;
}

// Reaching the step mints the command, as opening Add a computer does; a computer still waiting for its app from an
// earlier visit gets a fresh command instead of a second row beside it.
export const ComputerEnroll = ({ computers, onSetUp }: ComputerEnrollProps) => {
  const enroll = useEnrollComputer();
  const waiting = computers.find((c) => addedWithRunner(c) && !c.runner)?.id;
  const minted = useRef(false);
  useEffect(() => {
    if (minted.current) return;
    minted.current = true;
    enroll.mutate(waiting);
  }, [enroll, waiting]);

  const added = enroll.data?.computer;
  const computer = computers.find((c) => c.id === added?.id) ?? added;
  const paired = !!computer && pairedThroughRunner(computer);

  return (
    <>
      {enroll.isPending && <LoadingDisplay label="Making the command" />}
      {enroll.error && <ErrorDisplay error={enroll.error} title="Couldn't make the command." />}
      {enroll.data && computer && <AddComputerStep enrollment={enroll.data} computer={computer} />}
      <WelcomeFooter skip>
        {enroll.error && (
          <Button type="button" variant="outline" onClick={() => enroll.mutate(waiting)}>
            Try again
          </Button>
        )}
        <Button type="button" disabled={!paired} onClick={() => computer && onSetUp(computer.id)}>
          Set up
        </Button>
      </WelcomeFooter>
    </>
  );
};
