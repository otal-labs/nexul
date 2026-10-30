import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useFitText } from "@/hooks/useFitText";
import { cn } from "@/lib/utils";
import { fitTail, type Measure } from "@/utils/FitTextUtility";
import { splitTail } from "@/utils/RunnerFleetUtility";

interface RunnerNameProps {
  name: string;
  muted?: boolean;
}

const fitName = (name: string, width: number, measure: Measure) => fitTail(...splitTail(name), width, measure);

// The head is cut and the last dash segment stays, so two runners differing only at the end still read apart.
export const RunnerName = ({ name, muted = false }: RunnerNameProps) => {
  const [ref, shown] = useFitText<HTMLSpanElement>(name, fitName);
  const cut = shown !== name;
  return (
    <Tooltip open={cut ? undefined : false}>
      <TooltipTrigger asChild>
        <span ref={ref} className={cn("min-w-0 flex-1 overflow-hidden font-mono text-[13px] whitespace-nowrap", muted && "text-muted-foreground")}>
          {!cut && name}
          {cut && <span aria-hidden>{shown}</span>}
          {cut && <span className="sr-only">{name}</span>}
        </span>
      </TooltipTrigger>
      <TooltipContent className="font-mono">{name}</TooltipContent>
    </Tooltip>
  );
};
