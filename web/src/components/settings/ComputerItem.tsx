import { useState } from "react";

import { MoveComputerDialog, type MoveFrom } from "@/components/pairing/MoveComputerDialog";
import { ComputerRow } from "@/components/settings/ComputerRow";
import { RunnerComputerRow } from "@/components/settings/RunnerComputerRow";
import { useEnrollComputer } from "@/hooks/ComputerHooks";
import { addedWithRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface ComputerItemProps {
  computer: Computer;
  presence?: string | undefined;
}

// A computer added with its app reads by its lanes; one paired through a tunnel or by URL keeps its old row until it
// moves. The move dialog lives here, so the row turning into the app's row as the move lands never closes it.
export const ComputerItem = ({ computer, presence }: ComputerItemProps) => {
  const [moveFrom, setMoveFrom] = useState<MoveFrom>();
  const enroll = useEnrollComputer();
  const startMove = () => {
    setMoveFrom(computer.tunnel ? "tunnel" : "url");
    if (!computer.runner) enroll.mutate(computer.id);
  };
  const endMove = () => {
    setMoveFrom(undefined);
    enroll.reset();
  };

  return (
    <>
      {addedWithRunner(computer) && <RunnerComputerRow computer={computer} presence={presence} />}
      {!addedWithRunner(computer) && <ComputerRow computer={computer} presence={presence} onMove={startMove} />}
      <MoveComputerDialog computer={computer} from={moveFrom} enroll={enroll} onClose={endMove} />
    </>
  );
};
