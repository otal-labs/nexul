import { CircleX, Loader2 } from "lucide-react";

interface StepStatusMarkProps {
  live: boolean;
  failed: boolean;
}

// A spinner while the step runs, a failed mark once it failed, nothing otherwise; colour only on the failed icon.
export const StepStatusMark = ({ live, failed }: StepStatusMarkProps) => (
  <>
    {live && <Loader2 className="size-3 shrink-0 animate-spin text-muted-foreground motion-reduce:animate-none" role="img" aria-label="running" />}
    {failed && (
      <span className="inline-flex shrink-0 items-center gap-1 text-muted-foreground">
        <CircleX className="size-3.5 text-destructive" aria-hidden />
        failed
      </span>
    )}
  </>
);
