import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Doc, DocListItem } from "@/models/Doc";

export const getDocsKey = "getDocs";
export const getDocKey = "getDoc";

export const useFetchDocsByProject = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getDocsKey, projectId],
    queryFn: () => api.get<DocListItem[]>(`/api/docs?project_id=${encodeURIComponent(projectId ?? "")}`),
    enabled: !!projectId,
  });

export const useFetchDoc = (id: string | undefined) =>
  useQuery({
    queryKey: [getDocKey, id],
    queryFn: () => api.get<Doc>(`/api/docs/${id}`),
    enabled: !!id,
  });
