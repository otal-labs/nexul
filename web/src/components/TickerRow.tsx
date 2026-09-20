import { CheckCircle2, Circle, Loader2, XCircle } from "lucide-react";

export interface CheckOutcome {
  state: "idle" | "pending" | "ok" | "failed";
  message?: string;
}

interface TickerRowProps {
  label: string;
  why?: string | undefined;
  outcome: CheckOutcome;
}

// One ticker row: listed up front with why it's needed, then a spinner while its request runs, a tick, or a cross with the reason.
export const TickerRow = ({ label, why, outcome }: TickerRowProps) => (
  <li role="status" className="flex items-start gap-2 text-sm" data-state={outcome.state}>
    <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center">
      {outcome.state === "idle" && <Circle className="size-4 text-border" aria-hidden />}
      {outcome.state === "pending" && (
        <Loader2 className="size-4 animate-spin text-muted-foreground motion-reduce:animate-none" aria-hidden />
      )}
      {outcome.state === "ok" && (
        <CheckCircle2 className="size-4 text-success animate-in zoom-in-50 fade-in-0 duration-200 ease-out" aria-hidden />
      )}
      {outcome.state === "failed" && (
        <XCircle className="size-4 text-destructive animate-in zoom-in-50 fade-in-0 duration-200 ease-out" aria-hidden />
      )}
    </span>
    <span className="min-w-0">
      <span className={outcome.state === "failed" ? "text-destructive" : undefined}>{label}</span>
      {outcome.state === "failed" && outcome.message && (
        <span className="block text-xs text-destructive/80">{outcome.message}</span>
      )}
      {why && outcome.state !== "failed" && <span className="block text-xs text-muted-foreground">{why}</span>}
    </span>
  </li>
);
