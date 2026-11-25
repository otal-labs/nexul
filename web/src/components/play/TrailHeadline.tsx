import { formatElapsed } from "@nexul/client-core/agentSteps";
import { personLabel } from "@nexul/client-core/person";
import { isTrailActive, type TrailState } from "@nexul/client-core/trail";

import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { DialogDescription } from "@/components/ui/dialog";
import { useLiveTrailState } from "@/hooks/TrailHooks";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { type Trail } from "@/models/Trail";
import { formatClockTime, formatShortDate } from "@/utils/TimeUtility";

interface TrailHeadlineProps {
  trail: Trail;
}

const VIA_WORDS: Record<Exclude<Trail["via"], "automation">, string> = { web: "from the web", mcp: "over MCP" };

const stateWords = (state: TrailState, seconds: number) => {
  const took = formatElapsed(seconds);
  if (state === "starting") return { word: "Starting for", time: took };
  if (state === "running") return { word: "Running for", time: took };
  if (state === "waiting") return { word: "Waiting for an answer", time: "" };
  if (state === "done") return { word: "Done in", time: took };
  if (state === "failed") return { word: "Failed after", time: took };
  return { word: "Interrupted after", time: took };
};

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
        <span className="tabular-nums">{time ? `${word} ${time}` : word}</span>
      </span>
      {trail.via === "automation" && <span>Started by an automation</span>}
      {trail.via !== "automation" && (
        <span>
          Started by {personLabel(lookup(trail.starter_id))} {VIA_WORDS[trail.via]}
        </span>
      )}
      <time dateTime={trail.started_at} className="font-mono tabular-nums">
        {formatShortDate(trail.started_at)} {formatClockTime(trail.started_at)}
      </time>
    </DialogDescription>
  );
};
