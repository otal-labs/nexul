import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getInstallationsKey, getRepositoriesKey } from "@/hooks/RepositoryHooks";
import type { GitHubLinkStatus } from "@/models/GitHubLink";

export const getGitHubLinkKey = "github-link";

export const useFetchGitHubLink = () =>
  useQuery({
    queryKey: [getGitHubLinkKey],
    queryFn: async () => (await api.get<GitHubLinkStatus>("/api/auth/github-link")).data,
  });

// The person's repositories and installations were read with the token this forgets, so both lists go stale with it.
export const useDisconnectGitHub = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => api.delete("/api/auth/github-link"),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: [getGitHubLinkKey] }),
        client.invalidateQueries({ queryKey: [getRepositoriesKey] }),
        client.invalidateQueries({ queryKey: [getInstallationsKey] }),
      ]);
      toast.success("GitHub disconnected");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
