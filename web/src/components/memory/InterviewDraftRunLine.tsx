import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchInterviewDrafts, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useFetchInterviewAnswers, useFetchProjectInterviewTemplate } from "@/hooks/MemoryHooks";
import { draftRunCount } from "@/models/InterviewAnswer";
import { usePlayRunStore } from "@/stores/playRunStore";
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

// The latest drafting run as one line: its trail icon, its state, and what it drafted that still waits on the person.
export const InterviewDraftRunLine = ({ projectId }: InterviewDraftRunLineProps) => {
  const latest = useInterviewTrails(projectId).drafting?.[0];
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const { data: drafts } = useFetchInterviewDrafts(projectId);
  const { data: template } = useFetchProjectInterviewTemplate(projectId);
  const { data: answers } = useFetchInterviewAnswers(projectId);
  if (!latest || !state) return null;
  return (
    <p className="flex min-w-0 items-center gap-2 px-1.5 py-1 text-sm">
      <TrailStateIcon state={state} />
      <span className="shrink-0 font-medium">{STATE_LABEL[state]}</span>
      {template && answers && (
        <span className="truncate text-muted-foreground tabular-nums">
          · {draftRunCount(template.questions, answers, drafts ?? [], Date.parse(latest.started_at))}
        </span>
      )}
    </p>
  );
};
