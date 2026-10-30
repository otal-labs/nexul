import { useMemo, type ReactNode } from "react";

import { useFetchChatThreadIndicators } from "@/hooks/ChatHooks";
import { useActionState } from "@/hooks/useActionState";
import { BoardCardDataContext, CanMoveTicketsContext } from "@/hooks/useBoardCardData";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchLabelColors } from "@/hooks/LabelHooks";
import { useFetchBlockers } from "@/hooks/TicketLinkHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { useFetchActiveTrails } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface BoardCardProviderProps {
  projectId: string | undefined;
  children: ReactNode;
}

// Once per board, not per card: each query observer arms its own stale timer on mount, so per-card hooks stalled opening a big board.
export const BoardCardProvider = ({ projectId, children }: BoardCardProviderProps) => {
  const { data: project } = useFetchProject(projectId);
  const { data: ticketTypes } = useFetchProjectTicketTypes(projectId);
  const { data: labelColors } = useFetchLabelColors(projectId);
  const { data: threadIndicators } = useFetchChatThreadIndicators(projectId);
  const { data: statuses } = useFetchProjectStatuses(projectId);
  const { data: blockers } = useFetchBlockers();
  const { data: activeTrails } = useFetchActiveTrails(projectId);
  const { data: people } = useFetchWorkspacePeople(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const canMoveTickets = useActionState({ permission: "tickets:write", projectId }).state === "live";
  const prefix = project?.prefix ?? "";

  const data = useMemo(
    () => ({ prefix, ticketTypes, labelColors, threadIndicators, statuses, blockers, activeTrails, people }),
    [prefix, ticketTypes, labelColors, threadIndicators, statuses, blockers, activeTrails, people],
  );

  return (
    <CanMoveTicketsContext value={canMoveTickets}>
      <BoardCardDataContext value={data}>{children}</BoardCardDataContext>
    </CanMoveTicketsContext>
  );
};
