import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { defineQuery } from "@/lib/liveQuery";

export const getLabelColorsKey = "getLabelColors";

// A color change publishes no topic, so the board reads them again when it mounts or the app returns to the front.
const labelColorsQuery = defineQuery({
  key: getLabelColorsKey,
  fetch: requires("tickets:read", (projectId: string | undefined, labels: string[]) =>
    api.post<Record<string, string>>("/api/tickets/labels/colors", { project_id: projectId, labels }),
  ),
  refreshes: {},
});

// The board asks once with its own label set, and its cards read the answer from context.
export const useBoardLabelColors = (projectId: string | undefined) => {
  const { data: tickets } = useFetchTicketsByProject(projectId);
  const labels = useMemo(() => [...new Set(tickets?.flatMap((ticket) => ticket.labels ?? []))].sort(), [tickets]);
  return useQuery({ ...labelColorsQuery.options(projectId, labels), enabled: !!projectId && labels.length > 0 });
};
