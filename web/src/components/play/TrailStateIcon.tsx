import { CheckCircle2, CircleHelp, LoaderCircle, XCircle } from "lucide-react";

import { isTrailActive, type TrailState } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface TrailStateIconProps {
  state: TrailState;
  className?: string;
}

// Spinner while active, a question mark while waiting on the user, tick on done, cross on failed or interrupted;
// the only color on a trail row.
export const TrailStateIcon = ({ state, className }: TrailStateIconProps) => (
  <>
    {state === "waiting" && <CircleHelp className={cn("size-3.5 shrink-0 text-info", className)} role="img" aria-label="waiting" />}
    {isTrailActive(state) && state !== "waiting" && (
      <LoaderCircle className={cn("size-3.5 shrink-0 animate-spin text-warning", className)} role="img" aria-label={state} />
    )}
    {state === "done" && <CheckCircle2 className={cn("size-3.5 shrink-0 text-success", className)} role="img" aria-label="done" />}
    {(state === "failed" || state === "interrupted") && (
      <XCircle className={cn("size-3.5 shrink-0 text-destructive", className)} role="img" aria-label={state} />
    )}
  </>
);
