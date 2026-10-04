import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import type { Memory } from "@/models/Memory";
import type { TrailState } from "@/models/Trail";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface InterviewRunLineProps {
  projectId: string;
  memory: Memory | undefined;
  // Answers saved since the memory was last generated; any turns the done tick into the warning dot.
  changed: number;
}

const STATE_LABEL: Record<TrailState, string> = {
  starting: "Agent starting",
  running: "Agent working",
  waiting: "Agent asking",
  done: "Run finished",
  failed: "Run failed",
  interrupted: "Run stopped",
};

const lineFor = (state: TrailState | undefined, memory: Memory | undefined, changed: number): { label: string; detail: string } => {
  const written = memory ? `Written ${formatRelativeTime(memory.updated_at)}` : "";
  const ready = (state === undefined || state === "done") && memory;
  if (ready && changed > 0) return { label: "Memory ready", detail: `${changed} ${changed === 1 ? "answer" : "answers"} changed` };
  if (ready) return { label: "Memory ready", detail: written };
  if (state === undefined) return { label: "Not generated yet", detail: "" };
  return { label: STATE_LABEL[state], detail: state === "running" || state === "starting" ? "" : written };
};

// The latest interview run as one line: the trail's state icon, what it is doing, and when the memory was written.
export const InterviewRunLine = ({ projectId, memory, changed }: InterviewRunLineProps) => {
  const latest = useInterviewTrails(projectId).followUp?.[0];
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const stale = (state === undefined || state === "done") && !!memory && changed > 0;
  const icon = state ?? (memory ? "done" : undefined);
  const { label, detail } = lineFor(state, memory, changed);
  return (
    <p className="flex min-w-0 flex-1 basis-64 items-center gap-2 text-sm">
      {stale && (
        <span role="img" aria-label="out of date" className="flex size-3.5 shrink-0 items-center justify-center">
          <span className="size-2 rounded-full bg-warning" />
        </span>
      )}
      {!stale && icon && <TrailStateIcon state={icon} />}
      <span className="shrink-0 font-medium">{label}</span>
      {detail !== "" && <span className="truncate text-muted-foreground">· {detail}</span>}
    </p>
  );
};
