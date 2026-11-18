import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { workspacePath } from "@/models/Workspace";
import { useViewAsStore } from "@/stores/viewAsStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface ViewAsGrant {
  workspace_id: string;
  user_id: string;
  name: string;
  avatar_url: string;
}

interface ViewAsRequest {
  workspaceId: string;
  workspaceName: string;
  userId: string;
}

// The server checks the viewer may see what the person sees, then every later request carries the two headers.
export const useStartViewAs = () => {
  const client = useQueryClient();
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const { data: workspaces } = useFetchWorkspaces();
  return useMutation({
    mutationFn: requires("members:view_as", async ({ workspaceId, userId }: ViewAsRequest) =>
      (await api.post<ViewAsGrant>(`/api/workspaces/${encodeURIComponent(workspaceId)}/members/${encodeURIComponent(userId)}/view-as`)).data),
    onSuccess: (grant, { workspaceName }) => {
      const slug = workspaces?.find((workspace) => workspace.id === grant.workspace_id)?.slug ?? "";
      useViewAsStore.getState().start({
        workspaceId: grant.workspace_id,
        workspaceName,
        userId: grant.user_id,
        name: grant.name,
        avatarUrl: grant.avatar_url,
        returnTo: `${pathname}${search}`,
      });
      useWorkspaceStore.getState().selectWorkspace(grant.workspace_id, slug);
      client.clear();
      void navigate(slug ? workspacePath(slug, "/") : "/");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Back to the page the view started from; a view started before that was remembered goes back to Team.
export const useExitViewAs = () => {
  const client = useQueryClient();
  const navigate = useNavigate();
  return () => {
    const returnTo = useViewAsStore.getState().target?.returnTo ?? "/settings/team";
    useViewAsStore.getState().exit();
    client.clear();
    void navigate(returnTo);
  };
};
