import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import type { Memory } from "@/models/Memory";
import type { TrailState } from "@/models/Trail";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface InterviewRunLineProps {
  projectId: string;
  memory: Memory | undefined;
}

const STATE_LABEL: Record<TrailState, string> = {
  starting: "Agent starting",
  running: "Agent working",
  waiting: "Agent asking",
  done: "Run finished",
  failed: "Run failed",
  interrupted: "Run stopped",
};

const lineFor = (state: TrailState | undefined, memory: Memory | undefined): { label: string; detail: string } => {
  const written = memory ? `Written ${formatRelativeTime(memory.updated_at)}` : "";
  if ((state === undefined || state === "done") && memory) return { label: "Memory ready", detail: written };
  if (state === undefined) return { label: "Not generated yet", detail: "" };
  return { label: STATE_LABEL[state], detail: state === "running" || state === "starting" ? "" : written };
};

// The latest interview run as one line: the trail's state icon, what it is doing, and when the memory was written.
export const InterviewRunLine = ({ projectId, memory }: InterviewRunLineProps) => {
  const { data: trails } = useFetchTrails("interview", projectId);
  const latest = trails?.[0];
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const icon = state ?? (memory ? "done" : undefined);
  const { label, detail } = lineFor(state, memory);
  return (
    <p className="flex min-w-0 flex-1 items-center gap-2 text-sm">
      {icon && <TrailStateIcon state={icon} />}
      <span className="shrink-0 font-medium">{label}</span>
      {detail !== "" && <span className="truncate text-muted-foreground">· {detail}</span>}
    </p>
  );
};
