import { useState } from "react";

import { MoveComputerDialog } from "@/components/pairing/MoveComputerDialog";
import { LegacyComputerPane } from "@/components/agent/LegacyComputerPane";
import { RunnerComputerPane } from "@/components/agent/RunnerComputerPane";
import { useEnrollComputer } from "@/hooks/ComputerHooks";
import { useFetchPresence } from "@/hooks/PairingHooks";
import { addedWithRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface OwnComputerPaneProps {
  computer: Computer;
}

// A computer added with its app reads by its lanes; one paired before the app keeps its old pane until it moves. The
// move dialog lives here, so the pane turning into the app's pane as the move lands never closes it.
export const OwnComputerPane = ({ computer }: OwnComputerPaneProps) => {
  const presence = useFetchPresence();
  const [moving, setMoving] = useState(false);
  const enroll = useEnrollComputer();
  const startMove = () => {
    setMoving(true);
    if (!computer.runner) enroll.mutate(computer.id);
  };
  const endMove = () => {
    setMoving(false);
    enroll.reset();
  };

  return (
    <div>
      {addedWithRunner(computer) && <RunnerComputerPane computer={computer} presence={presence.data?.[computer.id]} />}
      {!addedWithRunner(computer) && <LegacyComputerPane computer={computer} onMove={startMove} />}
      <MoveComputerDialog computer={computer} open={moving} enroll={enroll} onClose={endMove} />
    </div>
  );
};
