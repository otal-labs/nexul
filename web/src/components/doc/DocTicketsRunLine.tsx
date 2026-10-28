import { Link } from "react-router";

import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useFetchApplicablePlays } from "@/hooks/PlayHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchTicketsByDoc } from "@/hooks/TicketHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath } from "@/models/Project";
import { usePlayRunStore } from "@/stores/playRunStore";
import { latestRunOf, type TrailState } from "@/models/Trail";

interface DocTicketsRunLineProps {
  workspaceId: string;
  docId: string;
  projectId: string;
}

const TO_TICKETS_KEY = "to-tickets-via-ai";

const stateLabel = (state: TrailState, written: number): string => {
  if (state === "done") return written === 1 ? "1 ticket written" : `${written} tickets written`;
  if (state === "failed") return "Writing tickets failed";
  if (state === "interrupted") return "Writing tickets stopped";
  return "Writing tickets";
};

// The doc's latest To tickets via AI run as one line: its state, the tickets it has written so far, and the way to the board.
export const DocTicketsRunLine = ({ workspaceId, docId, projectId }: DocTicketsRunLineProps) => {
  const wsPath = useWorkspacePath();
  const { data: plays } = useFetchApplicablePlays(workspaceId, projectId, "doc", undefined);
  const playId = plays?.find((p) => p.builtin_key === TO_TICKETS_KEY)?.id;
  const { data: trails } = useFetchTrails("doc", docId);
  const latest = playId ? latestRunOf(trails, playId) : undefined;
  const state = usePlayRunStore((s) => (latest ? (s.frames[latest.id]?.state ?? latest.state) : undefined));
  const { data: tickets } = useFetchTicketsByDoc(latest ? docId : undefined);
  const { data: project } = useFetchProject(latest ? projectId : undefined);
  if (!latest || !state) return null;
  const start = Date.parse(latest.started_at);
  const written = (tickets ?? []).filter((t) => Date.parse(t.created_at) >= start).length;
  return (
    <p className="flex min-w-0 items-center gap-2 text-sm">
      <TrailStateIcon state={state} />
      <span className="shrink-0 font-medium">{stateLabel(state, written)}</span>
      {state !== "done" && written > 0 && <span className="text-muted-foreground tabular-nums">· {written} so far</span>}
      {written > 0 && project && (
        <Link
          to={wsPath(boardPath(project))}
          className="text-muted-foreground underline-offset-4 transition-colors duration-150 ease-standard hover:text-foreground hover:underline"
        >
          · Open board
        </Link>
      )}
    </p>
  );
};
