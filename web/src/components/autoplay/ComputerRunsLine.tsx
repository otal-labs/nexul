import { ComputerRunsSelect } from "@/components/autoplay/ComputerRunsSelect";
import { useListComputers } from "@/hooks/PairingHooks";

// The tail closes the item: a comma, nothing before the last (which opens with its own "and"), and the full stop after it.
const tailOf = (index: number, count: number) => {
  if (index === count - 1) return ".";
  if (index === count - 2) return "";
  return ",";
};

// The viewer's own computers only: a computer's owner alone sets how many runs it takes, so it is theirs to change here whatever they may do to the play.
export const ComputerRunsLine = () => {
  const { data: computers } = useListComputers();

  return (
    computers &&
    computers.length > 0 && (
      <p className="text-xs leading-7 text-muted-foreground">
        At once,{" "}
        {computers.map((computer, i) => (
          <ComputerRunsSelect key={computer.id} computer={computer} lead={i === 0 ? "runs on" : "on"} tail={tailOf(i, computers.length)} and={i > 0 && i === computers.length - 1} />
        ))}
      </p>
    )
  );
};
