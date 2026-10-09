import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Doc, DocListItem } from "@/models/Doc";

export const getDocsKey = "getDocs";
export const getDocKey = "getDoc";

const docsQuery = defineQuery({
  key: getDocsKey,
  fetch: (projectId: string | undefined) => api.get<DocListItem[]>(`/api/docs?project_id=${encodeURIComponent(projectId ?? "")}`),
  refreshes: {
    "doc.created": { key: (p) => p.doc.project_id },
    "doc.updated": { key: (p) => p.doc.project_id },
    "doc.deleted": { key: (p) => p.project_id },
  },
});

export const useFetchDocsByProject = (projectId: string | undefined) =>
  useQuery({ ...docsQuery.options(projectId), enabled: !!projectId });

const docQuery = defineQuery({
  key: getDocKey,
  fetch: (id: string | undefined) => api.get<Doc>(`/api/docs/${id}`),
  refreshes: {
    "doc.updated": { record: (p) => p.doc.id },
    "doc.deleted": { record: (p) => p.id },
  },
});

export const useFetchDoc = (id: string | undefined) => useQuery({ ...docQuery.options(id), enabled: !!id });
