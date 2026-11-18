import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { useFetchInstallations } from "@/hooks/RepositoryHooks";
import type { IssueSync } from "@/models/IssueSync";

export const getIssueSyncsKey = "getIssueSyncs";

export const useFetchIssueSyncs = (projectId: string) =>
  useQuery({
    queryKey: [getIssueSyncsKey, projectId],
    queryFn: async () => (await api.get<IssueSync[]>(`/api/projects/${projectId}/issue-sync`)).data,
  });

interface SetIssueSyncInput {
  owner: string;
  repo: string;
  enabled: boolean;
}

// The switch is the feedback, so a save sends no success toast; turning it on links the open issues soon after.
export const useSetIssueSync = (projectId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("projects:write", async (input: SetIssueSyncInput) =>
      (await api.put<IssueSync>(`/api/projects/${projectId}/issue-sync`, input)).data),
    onSuccess: () => client.invalidateQueries({ queryKey: [getIssueSyncsKey, projectId] }),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Whether the App's installation on owner has not accepted the Issues permission; unknown reads as not missing.
export const useIssuesPermissionMissing = (owner: string) => {
  const { data: installations } = useFetchInstallations();
  return !!installations?.some((i) => i.account_login.toLowerCase() === owner.toLowerCase() && i.missing_permissions?.includes("issues"));
};
