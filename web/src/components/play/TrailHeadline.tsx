import { formatElapsed } from "@nexul/client-core/agentSteps";
import { personLabel } from "@nexul/client-core/person";

import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { DialogDescription } from "@/components/ui/dialog";
import { useLiveTrailState } from "@/hooks/TrailHooks";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { isTrailActive, type Trail, type TrailState } from "@/models/Trail";

interface TrailHeadlineProps {
  trail: Trail;
}

const VIA_WORDS: Record<Trail["via"], string> = { web: "from the web", mcp: "over MCP", automation: "by an automation" };

const stateWords = (state: TrailState, seconds: number) => {
  const took = formatElapsed(seconds);
  if (state === "starting") return { word: "Starting", time: "" };
  if (state === "running") return { word: "Running for", time: took };
  if (state === "waiting") return { word: "Waiting for an answer", time: "" };
  if (state === "done") return { word: "Done in", time: took };
  if (state === "failed") return { word: "Failed after", time: took };
  return { word: "Interrupted after", time: took };
};

const formatStart = (iso: string) => new Date(iso).toLocaleString(undefined, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });

// The dialog's meta line: how the run stands and for how long, then who started it, from where and when.
export const TrailHeadline = ({ trail }: TrailHeadlineProps) => {
  const state = useLiveTrailState(trail);
  const lookup = usePersonLookup(trail.workspace_id);
  const started = Date.parse(trail.started_at);
  const elapsed = useElapsedSeconds(started, isTrailActive(state));
  const seconds = trail.ended_at ? Math.max(0, Math.round((Date.parse(trail.ended_at) - started) / 1_000)) : elapsed;
  const { word, time } = stateWords(state, seconds);

  return (
    <DialogDescription className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      <span className="inline-flex items-center gap-1.5 text-foreground">
        <TrailStateIcon state={state} />
        {word}
        {time && <span className="tabular-nums">{time}</span>}
      </span>
      <span>
        Started by {personLabel(lookup(trail.starter_id))} {VIA_WORDS[trail.via]}
      </span>
      <time dateTime={trail.started_at} className="tabular-nums">
        {formatStart(trail.started_at)}
      </time>
    </DialogDescription>
  );
};
