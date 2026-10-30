import { CircleCheckIcon, CircleHelpIcon, LoaderCircleIcon, LockIcon } from "lucide-react";

import type { TrailState } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface DocStateIconProps {
  runState: TrailState | undefined;
  locked: boolean;
  questionsClosed: boolean;
  className?: string;
}

// A doc's one mark, a run before the lock before closed questions, or an empty slot of the same size so titles stay in one column.
export const DocStateIcon = ({ runState, locked, questionsClosed, className }: DocStateIconProps) => (
  <>
    {runState === "waiting" && <CircleHelpIcon className={cn("size-3.5 shrink-0 text-info", className)} role="img" aria-label="Play waiting for an answer" />}
    {runState !== undefined && runState !== "waiting" && (
      <LoaderCircleIcon className={cn("size-3.5 shrink-0 animate-spin text-warning motion-reduce:animate-none", className)} role="img" aria-label="Play running" />
    )}
    {runState === undefined && locked && <LockIcon className={cn("size-3.5 shrink-0 p-px text-muted-foreground", className)} role="img" aria-label="Locked" />}
    {runState === undefined && !locked && questionsClosed && <CircleCheckIcon className={cn("size-3.5 shrink-0 text-success", className)} role="img" aria-label="Questions closed" />}
    {runState === undefined && !locked && !questionsClosed && <span className={cn("size-3.5 shrink-0", className)} aria-hidden />}
  </>
);
