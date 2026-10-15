import { StepStatusMark } from "@/components/play/StepStatusMark";
import { cn } from "@/lib/utils";

interface TurnFileChipProps {
  label: string;
  live: boolean;
  failed: boolean;
  open: boolean;
  // onToggle is absent for a step with no detail to show.
  onToggle: (() => void) | undefined;
}

// One file or search of a files group, a button when its step has detail to open.
export const TurnFileChip = ({ label, live, failed, open, onToggle }: TurnFileChipProps) => (
  <li className="inline-flex max-w-full items-center gap-1.5">
    {!onToggle && <span className={cn("truncate", failed && "text-destructive")}>{label}</span>}
    {onToggle && (
      <button
        type="button"
        aria-expanded={open}
        onClick={onToggle}
        className={cn(
          "truncate rounded-sm transition-colors duration-150 ease-standard hover:text-foreground",
          open && "text-foreground",
          failed && "text-destructive",
        )}
      >
        {label}
      </button>
    )}
    <StepStatusMark live={live} failed={false} />
  </li>
);
