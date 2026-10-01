import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getHarnessResolveKey, useFetchHarnessProjects, useFetchHarnessProviders, useListComputers } from "@/hooks/PairingHooks";
import { findModel } from "@/models/ModelPick";
import type { ProjectLink, ProjectLinkFormData } from "@/models/Pairing";

export const getProjectLinksKey = "getProjectLinks";

// The caller's own links, one per project they linked and can still open (ADR 0102).
export const useFetchProjectLinks = () =>
  useQuery({
    queryKey: [getProjectLinksKey],
    queryFn: async () => (await api.get<{ links: ProjectLink[] }>("/api/pairing/projects")).data.links,
  });

// A link changes which computer the caller's turns in that project resolve to, so readiness refetches with it.
const invalidateLinks = (client: QueryClient) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getProjectLinksKey] }),
    client.invalidateQueries({ queryKey: [getHarnessResolveKey] }),
  ]);

export const useSetProjectLink = (projectId: string, projectName: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProjectLinkFormData) =>
      (await api.put<ProjectLink>(`/api/pairing/projects/${projectId}`, input)).data,
    onSuccess: async () => {
      await invalidateLinks(client);
      toast.success(`${projectName} linked`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useClearProjectLink = (projectId: string, projectName: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/pairing/projects/${projectId}`);
    },
    onSuccess: async () => {
      await invalidateLinks(client);
      toast.success(`${projectName} uses your defaults`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// "onik-desktop · nexul · Sonnet 5": names from the live harness where it answers, the stored ids where it doesn't.
export const useProjectLinkSummary = (link: ProjectLink | undefined): string => {
  const computerId = link?.computer_id ?? "";
  const { data: computers } = useListComputers();
  const { data: projects } = useFetchHarnessProjects(computerId);
  const { data: providers } = useFetchHarnessProviders(computerId);
  if (!link?.computer_id) return "Uses your defaults";

  const computer = computers?.find((c) => c.id === link.computer_id)?.name ?? link.computer_id;
  const project = projects?.find((p) => p.id === link.harness_project_id)?.title ?? link.harness_project_id;
  const pick = { provider: link.provider ?? "", model: link.model ?? "" };
  const providerName = providers?.find((p) => p.id === pick.provider)?.name ?? pick.provider;
  const model = findModel(providers ?? [], pick)?.name ?? (pick.model || providerName);
  return [computer, project, model].filter(Boolean).join(" · ");
};
