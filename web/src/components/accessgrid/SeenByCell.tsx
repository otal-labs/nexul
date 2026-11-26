import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { nameText } from "@/components/accessgrid/gridStyles";
import { cn } from "@/lib/utils";

interface SeenByCellProps {
  hiddenFrom: string[];
}

// Out of flow like a row's name, so a long list truncates inside the column instead of widening the grid.
const line = cn(nameText, "text-sm text-foreground");

// Who sees a public channel, in words: everyone, or everyone but the roles it is hidden from.
export const SeenByCell = ({ hiddenFrom }: SeenByCellProps) => {
  if (hiddenFrom.length === 0) {
    return <span className="block text-sm text-muted-foreground">Everyone</span>;
  }
  const text = `Everyone but ${hiddenFrom.join(", ")}`;
  return (
    <span className="relative block h-5">
      <Tooltip>
        <TooltipTrigger asChild>
          <span className={line}>{text}</span>
        </TooltipTrigger>
        <TooltipContent>{text}</TooltipContent>
      </Tooltip>
    </span>
  );
};
