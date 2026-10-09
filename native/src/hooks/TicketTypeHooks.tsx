import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";

export const getProjectTicketTypesKey = "getProjectTicketTypes";

// Only the fields the board row and ticket screen render; body_template is a web-only editing concern.
export interface TicketType {
  id: string;
  name: string;
  /** Owner-configured suggested-palette hue, "" when unset. */
  color: string;
}

const ticketTypesQuery = defineQuery({
  key: getProjectTicketTypesKey,
  fetch: (projectId: string | undefined) =>
    api.get<TicketType[]>(`/api/ticket-types?project_id=${encodeURIComponent(projectId ?? "")}`),
  refreshes: {},
});

export const useFetchProjectTicketTypes = (projectId: string | undefined) =>
  useQuery({ ...ticketTypesQuery.options(projectId), enabled: !!projectId });
