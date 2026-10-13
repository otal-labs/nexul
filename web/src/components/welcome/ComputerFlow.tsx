import { useState } from "react";

import { SetupDoneButton } from "@/components/pairing/SetupDoneButton";
import { SetupStep } from "@/components/pairing/SetupStep";
import { ComputerEnroll } from "@/components/welcome/ComputerEnroll";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { useListComputers } from "@/hooks/PairingHooks";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { pairedThroughRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface ComputerSetupProps {
  computerId: string;
}

// Continue waits for setup to be confirmed, then keeps the models and folder the way the dialog's Done does.
const ComputerSetup = ({ computerId }: ComputerSetupProps) => {
  const { marks, next } = useWelcomeFlow();
  const { data: computers } = useListComputers();
  const confirmed = marks.computer === "done";
  const computer = computers?.find((c) => c.id === computerId);
  return (
    <>
      {computer && <SetupStep computer={computer} />}
      <WelcomeFooter skip={!confirmed}>
        {confirmed && <SetupDoneButton computerId={computerId} label="Continue" onDone={next} />}
        {!confirmed && <ContinueButton disabled />}
      </WelcomeFooter>
    </>
  );
};

interface ComputerFlowProps {
  computers: Computer[];
}

// The Add a computer dialog's two steps in place: the command beside its three checks, then setup. A computer paired
// on an earlier visit goes straight to setup, since the step is left only while its setup isn't confirmed.
export const ComputerFlow = ({ computers }: ComputerFlowProps) => {
  const [settingUp, setSettingUp] = useState(() => computers.find(pairedThroughRunner)?.id);
  return (
    <>
      {settingUp && <ComputerSetup computerId={settingUp} />}
      {!settingUp && <ComputerEnroll computers={computers} onSetUp={setSettingUp} />}
    </>
  );
};
