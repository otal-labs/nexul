import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { AlreadyDone } from "@/components/welcome/AlreadyDone";
import { ComputerFlow } from "@/components/welcome/ComputerFlow";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { useListComputers } from "@/hooks/PairingHooks";
import { useDoneOnArrival } from "@/hooks/useWelcomeFlow";
import { pairedThroughRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

const computerLabel = (computers: Computer[]) => {
  const paired = computers.find(pairedThroughRunner) ?? computers.find((c) => c.server_url !== "");
  return paired?.name || paired?.facts?.hostname || "Your computer";
};

const ComputerReady = ({ computers }: { computers: Computer[] }) => (
  <>
    <AlreadyDone title={`${computerLabel(computers)} is paired and set up`}>Add another computer from Your settings, under Computers.</AlreadyDone>
    <WelcomeFooter skip={false}>
      <ContinueButton />
    </WelcomeFooter>
  </>
);

export const ComputerStep = () => {
  const arrivedDone = useDoneOnArrival("computer");
  const { data: computers, isPending, error } = useListComputers();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && arrivedDone && <ComputerReady computers={computers} />}
      {computers && !arrivedDone && <ComputerFlow computers={computers} />}
    </>
  );
};
