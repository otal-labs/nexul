import { formatUpdatedAgo } from "@/components/doc/docTime";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { needAnswers, type PlayRunHealth } from "@/models/Play";
import { cn } from "@/lib/utils";

interface PlayHealthLineProps {
  health: PlayRunHealth;
}

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

// What the play's runs say, busiest first; "last … ago" is left out while a run is in flight, where it would be that run's own start.
export const PlayHealthLine = ({ health }: PlayHealthLineProps) => {
  const lastFailed = health.last_run?.state === "failed";
  const inFlight = health.running > 0;

  const last = health.last_run;

  return (
    <>
      {!last && <p className="font-mono text-[12px] text-muted-foreground">Not run yet</p>}
      {last && (
        // Each figure leads with its dot, which a wrapped line pushes into the clipped margin, so no line starts or ends on one.
        <p className="overflow-hidden font-mono text-[12px] text-muted-foreground tabular-nums">
          <span className="-ml-5 flex flex-wrap items-center gap-y-1">
            {inFlight && <Figure state="running" text={`${health.running} running`} />}
            {health.waiting > 0 && <Figure state="waiting" text={needAnswers(health.waiting)} />}
            {health.failed_7d > 0 && <Figure state={lastFailed ? "failed" : undefined} text={`${health.failed_7d} failed this week`} />}
            <Figure text={`${plural(health.runs_7d, "run")} this week`} />
            {!inFlight && <Figure text={`last ${formatUpdatedAgo(last.started_at)}`} />}
          </span>
        </p>
      )}
    </>
  );
};

const Figure = ({ state, text }: { state?: "running" | "waiting" | "failed" | undefined; text: string }) => (
  <span className="inline-flex items-center gap-1 whitespace-nowrap before:w-4 before:text-center before:text-muted-foreground/60 before:content-['·']">
    {state && <TrailStateIcon state={state} className="size-3 -translate-y-px" />}
    <span className={cn(state && "text-foreground")}>{text}</span>
  </span>
);
