import { ComputerRow } from "@/components/settings/ComputerRow";
import { RunnerComputerRow } from "@/components/settings/RunnerComputerRow";
import { addedWithRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

interface ComputerItemProps {
  computer: Computer;
  presence?: string | undefined;
}

// A computer added with its app reads by its lanes; one paired through a tunnel or by URL keeps its old row until it moves.
export const ComputerItem = ({ computer, presence }: ComputerItemProps) => (
  <>
    {addedWithRunner(computer) && <RunnerComputerRow computer={computer} presence={presence} />}
    {!addedWithRunner(computer) && <ComputerRow computer={computer} presence={presence} />}
  </>
);
