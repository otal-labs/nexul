import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { useFetchInstallations } from "@/hooks/RepositoryHooks";
import { IssueSyncMode, type IssueSync, type IssueSyncChange } from "@/models/IssueSync";

export const getIssueSyncsKey = "getIssueSyncs";

export const useFetchIssueSyncs = (projectId: string) =>
  useQuery({
    queryKey: [getIssueSyncsKey, projectId],
    queryFn: async () => (await api.get<IssueSync[]>(`/api/projects/${projectId}/issue-sync`)).data,
    enabled: !!projectId,
  });

// Optimistic: the switch is the feedback, so a failure puts it back and says why.
export const useSetIssueSync = (projectId: string) => {
  const client = useQueryClient();
  const key = [getIssueSyncsKey, projectId];
  return useMutation({
    mutationFn: requires("projects:write", async (change: IssueSyncChange) =>
      (await api.put<IssueSync>(`/api/projects/${projectId}/issue-sync`, change)).data),
    onMutate: async (change) => {
      await client.cancelQueries({ queryKey: key });
      const before = client.getQueryData<IssueSync[]>(key);
      const others = (before ?? []).filter((s) => s.owner !== change.owner || s.repo !== change.repo);
      const current = before?.find((s) => s.owner === change.owner && s.repo === change.repo);
      const next: IssueSync = {
        project_id: projectId, owner: change.owner, repo: change.repo,
        label: change.label ?? current?.label ?? "nexul", mode: change.mode ?? current?.mode ?? IssueSyncMode.Label,
        enabled: change.enabled ?? current?.enabled ?? false,
      };
      client.setQueryData<IssueSync[]>(key, [...others, next]);
      return { before };
    },
    onError: (error, _change, context) => {
      client.setQueryData(key, context?.before);
      toast.error(errorMessage(error));
    },
    onSettled: () => client.invalidateQueries({ queryKey: key }),
  });
};

// The owners whose App installation has not accepted the Issues permission; an installation it can't read counts as fine.
export const useIssuesPermissionMissing = (owners: readonly string[]): string[] => {
  const { data: installations } = useFetchInstallations();
  const lacking = new Set(
    (installations ?? []).filter((i) => i.missing_permissions?.includes("issues")).map((i) => i.account_login.toLowerCase()),
  );
  return owners.filter((owner) => lacking.has(owner.toLowerCase()));
};
