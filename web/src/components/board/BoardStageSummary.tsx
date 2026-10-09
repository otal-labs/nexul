import { useLayoutEffect, useMemo, useRef } from "react";

import { playStageBar } from "@/components/board/stageBarMotion";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTickets } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import { STATUS_STAGES, StatusKind } from "@/models/Status";

// Backlog is work not yet started, so its share reads quieter than the stages in motion.
const stageBar = (kind: StatusKind, dot: string) => cn(dot, kind === StatusKind.Backlog && "opacity-50");

interface BoardStageSummaryProps {
  projectId: string | undefined;
  className?: string;
}

// Where the project's work stands: one slim bar split by stage, every ticket counted whatever the filters show.
export const BoardStageSummary = ({ projectId, className }: BoardStageSummaryProps) => {
  const { data: tickets } = useFetchTickets();
  const { data: statuses } = useFetchProjectStatuses(projectId);
  const stages = useMemo(() => {
    if (!tickets || !statuses) return [];
    const kindOf = new Map(statuses.map((s) => [s.id, s.kind]));
    const mine = tickets.filter((t) => t.project_id === projectId);
    return STATUS_STAGES.filter((stage) => statuses.some((s) => s.kind === stage.kind)).map((stage) => ({
      ...stage,
      count: mine.filter((t) => kindOf.get(t.status) === stage.kind).length,
    }));
  }, [tickets, statuses, projectId]);
  const total = stages.reduce((sum, s) => sum + s.count, 0);
  const ref = useRef<HTMLDivElement>(null);
  const counts = stages.map((s) => s.count).join(",");
  useLayoutEffect(() => playStageBar(ref.current), [counts]);
  if (total === 0) return null;

  return (
    <div ref={ref} className={cn("space-y-2", className)} role="group" aria-label="Tickets by stage">
      <div data-stage-bar className="flex h-1.5 w-full gap-0.5" aria-hidden>
        {stages.filter((s) => s.count > 0).map((s) => (
          <span key={s.kind} data-stage-segment={s.kind} className={cn("h-full origin-left rounded-full", stageBar(s.kind, s.dot))} style={{ flexGrow: s.count }} />
        ))}
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {stages.map((s) => (
          <li key={s.kind} className="flex items-center gap-1.5">
            <span className={cn("size-1.5 rounded-full", stageBar(s.kind, s.dot))} aria-hidden />
            {s.label}
            <span data-stage-count={s.kind} className="font-mono tabular-nums text-foreground">{s.count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
};
