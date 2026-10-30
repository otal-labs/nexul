import { cn } from "@/lib/utils";
import type { Computer } from "@/models/Pairing";

interface ComputerLoadProps {
  computer: Computer;
}

// How many play runs are going on the computer against how many it takes (ADR 0166).
export const ComputerLoad = ({ computer }: ComputerLoadProps) => {
  const full = computer.running >= computer.runs_at_once;
  return (
    <span
      className={cn("shrink-0 translate-y-px text-xs tabular-nums", computer.running > 0 ? "text-foreground" : "text-muted-foreground")}
      title={full ? "Full: new runs wait or go to your next computer" : undefined}
    >
      {computer.running} of {computer.runs_at_once} play runs
    </span>
  );
};
