import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { MembersList } from "@/models/Member";

export const getWorkspaceMembersKey = "getWorkspaceMembers";

// 403s for a viewer without members:write, so direct navigation gets the server's real answer.
export const useFetchWorkspaceMembers = (workspaceId: string) =>
  useQuery({
    queryKey: [getWorkspaceMembersKey, workspaceId],
    queryFn: async () => (await api.get<MembersList>(`/api/workspaces/${workspaceId}/members`)).data,
    enabled: workspaceId !== "",
  });

