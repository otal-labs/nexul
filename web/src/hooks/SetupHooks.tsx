import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AxiosError } from "axios";

import { api, joinAPIURL } from "@/api/client";
import { getBootstrapStatusKey } from "@/hooks/AuthHooks";
import { useSetupPassStore } from "@/stores/setupPassStore";
import type { GitHubManifestStart, PublicAddress, SetupPass } from "@/models/Setup";
import { retry } from "@/utils/RetryUtility";

const FINISH_RETRY_MS = 5000;
export const getPublicAddressKey = "getPublicAddress";

// Errors stay inline on the code screen (wrong code, throttled, already set up), so no toast here.
export const useUnlockSetup = () => {
  const client = useQueryClient();
  const unlock = useSetupPassStore((s) => s.unlock);
  return useMutation({
    mutationFn: async (code: string) => (await api.post<SetupPass>("/api/setup/unlock", { code })).data,
    onSuccess: (pass, code) => unlock(pass, code),
    onError: async (error) => {
      if ((error as AxiosError).response?.status === 409) {
        await client.invalidateQueries({ queryKey: [getBootstrapStatusKey] });
      }
    },
  });
};

// Success flips bootstrap-status to a stored instance URL, which moves setup on to the handoff or GitHub step.
export const useSetInstanceUrl = () => {
  const client = useQueryClient();
  return useMutation({
    // attempts > 1 covers a certificate still being issued: the server's HTTPS check fails until it lands.
    mutationFn: async ({ url, attempts }: { url: string; attempts: number }) =>
      retry(attempts, FINISH_RETRY_MS, async () =>
        (await api.put<{ instance_url: string }>("/api/setup/instance-url", { url })).data,
      ),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getBootstrapStatusKey] });
    },
  });
};

export const useFetchPublicAddress = () =>
  useQuery({
    queryKey: [getPublicAddressKey],
    queryFn: async () => (await api.get<PublicAddress>("/api/setup/public-address")).data,
    staleTime: Infinity,
  });

export const useStartGitHubManifest = () =>
  useMutation({ mutationFn: async () => (await api.post<GitHubManifestStart>("/api/setup/github-app/start")).data });

export const useCompleteGitHubManifest = () =>
  useMutation({
    mutationFn: async (callback: { code: string; state: string }) => {
      await api.post("/api/setup/github-app/callback", callback);
    },
    onSuccess: () => window.location.assign(joinAPIURL("/auth/github")),
  });
