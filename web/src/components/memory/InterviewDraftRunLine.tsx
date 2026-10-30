import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchInterviewSources, useInterviewPlays, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useActiveTrail } from "@/hooks/TrailHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import { lastDraftedAt, sourceChange } from "@/models/InterviewSource";
import type { TrailState } from "@/models/Trail";

interface InterviewDraftRunLineProps {
  projectId: string;
}

const STATE_LABEL: Record<TrailState, string> = {
  starting: "Drafting",
  running: "Drafting",
  waiting: "Drafting",
  done: "Drafts ready",
  failed: "Drafting failed",
  interrupted: "Drafting stopped",
};

// The latest drafting run's state and how many sources changed after it, and why Draft answers is off while another run goes.
export const InterviewDraftRunLine = ({ projectId }: InterviewDraftRunLineProps) => {
  const drafting = useInterviewTrails(projectId).drafting;
  const play = useInterviewPlays(projectId).draft;
  const active = useActiveTrail("interview", projectId);
  const latest = drafting?.[0];
  const { data: sources } = useFetchInterviewSources(projectId);
  const since = drafting && lastDraftedAt(drafting);
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const changed = state === "done" ? (sources ?? []).filter((s) => sourceChange(s, since) !== null).length : 0;
  const waits = !!play && !!active && active.play_id !== play.id && !!sources?.some((s) => s.stance === "follow");
  const line = [
    state && STATE_LABEL[state],
    changed > 0 && `${changed} ${changed === 1 ? "source" : "sources"} changed since drafting`,
    waits && "Draft answers waits for the current run",
  ].filter(Boolean).join(" · ");
  if (line === "") return null;
  return (
    <p className="flex min-w-0 items-start gap-2 px-1.5 text-xs text-muted-foreground">
      {state && <TrailStateIcon state={state} className="mt-px" />}
      <span className="min-w-0 text-pretty">{line}</span>
    </p>
  );
};
