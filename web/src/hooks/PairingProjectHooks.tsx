import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHarnessResolveKey, useFetchHarnessProjects, useFetchHarnessProviders, useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";
import { findModel } from "@/models/ModelPick";
import { canChooseRunLocation, START_IN_SHORT_LABELS, type ProjectLink, type ProjectLinkFormData, type ReadinessLink } from "@/models/Pairing";

export const getProjectLinksKey = "getProjectLinks";

// The caller's own links, one per project they linked and can still open (ADR 0102).
export const useFetchProjectLinks = () =>
  useQuery({
    queryKey: [getProjectLinksKey],
    queryFn: async () => (await api.get<{ links: ProjectLink[] }>("/api/pairing/projects")).data.links,
  });

// What blocks a play in the project and where it is fixed: no message when one can start; undefined until readiness and
// the caller's links have loaded.
export const useRunBlock = (projectId: string): { message: string; link?: ReadinessLink } | undefined => {
  const readiness = useHarnessReadiness(projectId);
  const links = useFetchProjectLinks();
  if (readiness === undefined || links.isPending) return undefined;
  if (readiness.state === "ready") return { message: "" };
  const linked = links.data?.some((l) => l.project_id === projectId && !!l.computer_id) ?? false;
  if (canChooseRunLocation(readiness, linked)) return { message: "" };
  return readiness;
};

// Why no play can start in the project, "" when one can; undefined until readiness and the caller's links have loaded.
export const useRunBlockedReason = (projectId: string): string | undefined => useRunBlock(projectId)?.message;

// A link changes which computer the caller's turns in that project resolve to, so readiness refetches with it.
export const invalidateProjectLinks = (client: QueryClient) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getProjectLinksKey] }),
    client.invalidateQueries({ queryKey: [getHarnessResolveKey] }),
  ]);

export const useSetProjectLink = (projectId: string, projectName: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async (input: ProjectLinkFormData) =>
      (await api.put<ProjectLink>(`/api/pairing/projects/${projectId}`, input)).data),
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
    mutationFn: requires("none", async () => {
      await api.delete(`/api/pairing/projects/${projectId}`);
    }),
    onSuccess: async () => {
      await invalidateProjectLinks(client);
      toast.success(`${projectName} uses your defaults`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// A link's computer and T3 project by name, never by id: an id is no name, so one that no longer resolves reads as unknown.
export const useHarnessPlaceNames = (computerId: string, harnessProjectId: string) => {
  const computers = useListComputers();
  const projects = useFetchHarnessProjects(computerId);
  return {
    computer: computers.data?.find((c) => c.id === computerId)?.name ?? "Unknown computer",
    project: projects.data?.find((p) => p.id === harnessProjectId)?.title ?? "Unknown project",
    isPending: computers.isPending || (computerId !== "" && projects.isPending),
  };
};

// "onik-desktop · nexul · Sonnet 5 · New worktree", or undefined while the names load.
export const useProjectLinkSummary = (link: ProjectLink | undefined): string | undefined => {
  const computerId = link?.computer_id ?? "";
  const { computer, project, isPending } = useHarnessPlaceNames(computerId, link?.harness_project_id ?? "");
  const { data: providers } = useFetchHarnessProviders(computerId);
  if (!link?.computer_id) return "Uses your defaults";
  if (isPending) return undefined;

  const pick = { provider: link.provider ?? "", model: link.model ?? "" };
  const providerName = providers?.find((p) => p.id === pick.provider)?.name ?? pick.provider;
  const model = findModel(providers ?? [], pick)?.name ?? (pick.model || providerName);
  const startIn = link.start_in && START_IN_SHORT_LABELS[link.start_in];
  return [computer, link.harness_project_id && project, model, startIn].filter(Boolean).join(" · ");
};
