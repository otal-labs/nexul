import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchInterviewSources, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
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

// The latest drafting run's state and how many sources changed after it; its drafts are counted in the progress line.
export const InterviewDraftRunLine = ({ projectId }: InterviewDraftRunLineProps) => {
  const drafting = useInterviewTrails(projectId).drafting;
  const latest = drafting?.[0];
  const { data: sources } = useFetchInterviewSources(projectId);
  const since = drafting && lastDraftedAt(drafting);
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const changed = state === "done" ? (sources ?? []).filter((s) => sourceChange(s, since) !== null).length : 0;
  if (!latest || !state) return null;
  return (
    <p className="flex min-w-0 items-center gap-2 px-1.5 text-xs text-muted-foreground">
      <TrailStateIcon state={state} />
      <span className="truncate">
        {STATE_LABEL[state]}
        {changed > 0 && ` · ${changed} ${changed === 1 ? "source" : "sources"} changed since drafting`}
      </span>
    </p>
  );
};
