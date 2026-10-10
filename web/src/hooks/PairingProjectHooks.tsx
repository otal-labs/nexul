import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getHarnessResolveKey, useFetchHarnessProjects, useFetchHarnessProviders, useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";
import { findModel } from "@/models/ModelPick";
import { canChooseRunLocation, START_IN_SHORT_LABELS, type ProjectLink, type ProjectLinkFormData } from "@/models/Pairing";

export const getProjectLinksKey = "getProjectLinks";

// The caller's own links, one per project they linked and can still open (ADR 0102).
export const useFetchProjectLinks = () =>
  useQuery({
    queryKey: [getProjectLinksKey],
    queryFn: async () => (await api.get<{ links: ProjectLink[] }>("/api/pairing/projects")).data.links,
  });

// Why no play can start in the project, "" when one can; undefined until readiness and the caller's links have loaded.
export const useRunBlockedReason = (projectId: string): string | undefined => {
  const readiness = useHarnessReadiness(projectId);
  const links = useFetchProjectLinks();
  if (readiness === undefined || links.isPending) return undefined;
  if (readiness.state === "ready") return "";
  const linked = links.data?.some((l) => l.project_id === projectId && !!l.computer_id) ?? false;
  return canChooseRunLocation(readiness, linked) ? "" : readiness.message;
};

// A link changes which computer the caller's turns in that project resolve to, so readiness refetches with it.
export const invalidateProjectLinks = (client: QueryClient) =>
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
      await invalidateProjectLinks(client);
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
      await invalidateProjectLinks(client);
      toast.success(`${projectName} uses your defaults`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// "onik-desktop · nexul · Sonnet 5 · New worktree": names from the live harness where it answers, the stored ids where it doesn't.
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
  const startIn = link.start_in && START_IN_SHORT_LABELS[link.start_in];
  return [computer, project, model, startIn].filter(Boolean).join(" · ");
};
