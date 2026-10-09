import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { BoardStatus } from "@/models/Status";

export const getProjectStatusesKey = "getProjectStatuses";

// Already ordered by kind then position (internal/platform/storage/queries/statuses.sql), the board's column order.
const statusesQuery = defineQuery({
  key: getProjectStatusesKey,
  fetch: (projectId: string | undefined) =>
    api.get<BoardStatus[]>(`/api/statuses?project_id=${encodeURIComponent(projectId ?? "")}`),
  refreshes: {
    "status.created": { key: (p) => p.status.project_id },
    "status.updated": { key: (p) => p.status.project_id },
    "status.deleted": { key: (p) => p.status.project_id },
  },
});

export const useFetchProjectStatuses = (projectId: string | undefined) =>
  useQuery({ ...statusesQuery.options(projectId), enabled: !!projectId });
