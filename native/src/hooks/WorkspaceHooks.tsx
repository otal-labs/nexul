import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { api } from "@/api/client";
import { AREA_PERMISSION, type Area } from "@/models/Access";
import { projectPermissions, type MyWorkspaceInfo, type Workspace } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const getWorkspacesKey = "getWorkspaces";

export const useFetchWorkspaces = () =>
  useQuery({
    queryKey: [getWorkspacesKey],
    queryFn: () => api.get<Workspace[]>("/api/workspaces"),
  });

// The selected workspace is where every tab's data belongs, since the app only shows one workspace at a time.
export const useSelectedWorkspace = (): Workspace | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data } = useFetchWorkspaces();
  return data?.find((workspace) => workspace.id === selectedWorkspaceId);
};

export const useCurrentWorkspaceId = (): string | undefined => useSelectedWorkspace()?.id;

export const getMyRoleKey = "getMyRole";

export const useFetchMyRole = () => {
  const workspaceId = useCurrentWorkspaceId();
  return useQuery({
    queryKey: [getMyRoleKey, workspaceId],
    queryFn: () => api.get<MyWorkspaceInfo>(`/api/workspaces/${workspaceId}/me`),
    enabled: !!workspaceId,
  });
};

// Undefined until permissions first arrive; isFetched, since a failed read refetches as pending and must not blink.
// Answers for projectId as the server does for a Restricted member (ADR 0097); with none, for any project.
export const useAreaAccess = (projectId?: string): ((area: Area) => boolean) | undefined => {
  const { data, isFetched } = useFetchMyRole();
  if (!isFetched) return undefined;
  const held = projectPermissions(data, projectId);
  return (area) => held.includes(AREA_PERMISSION[area]);
};

// F5 exception: repairs an empty or stale selection so the switcher and every workspace-scoped screen has one to read.
export const useEnsureWorkspaceSelected = () => {
  const { data } = useFetchWorkspaces();
  useEffect(() => {
    if (!data) return;
    const { selectedWorkspaceId, selectWorkspace } = useWorkspaceStore.getState();
    const stillMember = data.some((workspace) => workspace.id === selectedWorkspaceId);
    if (!stillMember) selectWorkspace(data[0]?.id ?? "");
  }, [data]);
};
