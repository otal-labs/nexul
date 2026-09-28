import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";

export const getProjectTicketTypesKey = "getProjectTicketTypes";

// Only the fields the board row and ticket screen render; body_template is a web-only editing concern.
export interface TicketType {
  id: string;
  name: string;
  /** Owner-configured suggested-palette hue, "" when unset. */
  color: string;
}

export const useFetchProjectTicketTypes = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectTicketTypesKey, projectId],
    queryFn: () => api.get<TicketType[]>(`/api/ticket-types?project_id=${encodeURIComponent(projectId ?? "")}`),
    enabled: !!projectId,
  });
