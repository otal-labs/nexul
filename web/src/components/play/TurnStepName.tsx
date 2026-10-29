import { StepStatusMark } from "@/components/play/StepStatusMark";
import { cn } from "@/lib/utils";

interface TurnStepNameProps {
  label: string;
  live: boolean;
  failed: boolean;
  open: boolean;
  // onToggle is absent for a step with no detail to show.
  onToggle: (() => void) | undefined;
}

// One file, search or command of a listed group, a button when its step has detail to open.
export const TurnStepName = ({ label, live, failed, open, onToggle }: TurnStepNameProps) => (
  <li className="inline-flex max-w-full min-w-0 items-center gap-1.5">
    {!onToggle && <span className={cn("truncate", failed && "text-destructive")}>{label}</span>}
    {onToggle && (
      <button
        type="button"
        aria-expanded={open}
        onClick={onToggle}
        title={label}
        className={cn(
          "truncate rounded-sm text-left transition-colors duration-150 ease-standard hover:text-foreground",
          open && "text-foreground",
          failed && "text-destructive",
        )}
      >
        {label}
      </button>
    )}
    <StepStatusMark live={live} failed={failed} />
  </li>
);
