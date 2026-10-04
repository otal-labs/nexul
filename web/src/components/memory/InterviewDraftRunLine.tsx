import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchInterviewDrafts, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
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

// The latest drafting run as one line: its trail icon, its state, and how many template questions it has drafted so far.
export const InterviewDraftRunLine = ({ projectId }: InterviewDraftRunLineProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const latest = useInterviewTrails(projectId).drafting?.[0];
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const { data: drafts } = useFetchInterviewDrafts(projectId);
  const { data: template } = useFetchInterviewTemplate(workspaceId);
  if (!latest || !state) return null;
  const questions = new Set(template?.questions.map((q) => q.text.trim()));
  const since = Date.parse(latest.started_at);
  const drafted = (drafts ?? []).filter((d) => questions.has(d.question) && Date.parse(d.drafted_at) >= since).length;
  return (
    <p className="flex min-w-0 items-center gap-2 px-1.5 py-1 text-sm">
      <TrailStateIcon state={state} />
      <span className="shrink-0 font-medium">{STATE_LABEL[state]}</span>
      {template && (
        <span className="truncate text-muted-foreground tabular-nums">
          · {drafted} of {template.questions.length} drafted
        </span>
      )}
    </p>
  );
};
