import { cn } from "@/lib/utils";
import { splitTail } from "@/utils/RunnerFleetUtility";

interface RunnerNameProps {
  name: string;
  muted?: boolean;
}

// The head truncates and the last dash segment stays, so two runners differing only at the end still read apart.
export const RunnerName = ({ name, muted = false }: RunnerNameProps) => {
  const [head, tail] = splitTail(name);
  return (
    <span title={name} className={cn("flex min-w-0 font-mono text-[13px]", muted && "text-muted-foreground")}>
      <span className="truncate">{head}</span>
      {tail && <span className="shrink-0">{tail}</span>}
    </span>
  );
};
