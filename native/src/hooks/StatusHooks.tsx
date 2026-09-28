import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { BoardStatus } from "@/models/Status";

export const getProjectStatusesKey = "getProjectStatuses";

// Already ordered by kind then position (internal/platform/storage/queries/statuses.sql), the board's column order.
export const useFetchProjectStatuses = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectStatusesKey, projectId],
    queryFn: () => api.get<BoardStatus[]>(`/api/statuses?project_id=${encodeURIComponent(projectId ?? "")}`),
    enabled: !!projectId,
  });
