import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { getHarnessResolveKey, useFetchHarnessProjects, useFetchHarnessProviders, useHarnessReadiness, useListComputers } from "@/hooks/PairingHooks";
import { findModel } from "@/models/ModelPick";
import { canChooseRunLocation, linkSaysSomething, START_IN_SHORT_LABELS, type ProjectLink, type ProjectLinkFormData, type ReadinessLink } from "@/models/Pairing";

export const getProjectLinksKey = "getProjectLinks";

// The caller's own links, one per project they linked and can still open (ADR 0102).
export const useFetchProjectLinks = () =>
  useQuery({
    queryKey: [getProjectLinksKey],
    queryFn: async () => (await api.get<{ links: ProjectLink[] }>("/api/pairing/projects")).data.links,
  });

// What blocks a play in the project and where it is fixed: no message when one can start, or when it would ask where;
// undefined until readiness has loaded.
export const useRunBlock = (projectId: string): { message: string; link?: ReadinessLink } | undefined => {
  const readiness = useHarnessReadiness(projectId);
  if (readiness === undefined) return undefined;
  if (readiness.state === "ready" || canChooseRunLocation(readiness)) return { message: "" };
  return readiness;
};

// Why no play can start in the project, "" when one can; undefined until readiness has loaded.
export const useRunBlockedReason = (projectId: string): string | undefined => useRunBlock(projectId)?.message;

// A link changes which computer the caller's turns in that project land on, so readiness refetches with it.
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
      toast.success(`${projectName} saved`);
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
      toast.success(`${projectName} follows your computers and defaults`);
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

// "Follows your computers · Sonnet 5 · New worktree", or the project's own computers by name; undefined while names load.
export const useProjectLinkSummary = (link: ProjectLink | undefined): string | undefined => {
  const computers = useListComputers();
  const firstComputer = link?.computers[0] ?? computers.data?.[0]?.id ?? "";
  const { data: providers, isPending } = useFetchHarnessProviders(firstComputer);
  if (!linkSaysSomething(link)) return "Follows your computers and defaults";
  if (computers.isPending || (link?.model && firstComputer !== "" && isPending)) return undefined;

  const names = link?.computers.map((id) => computers.data?.find((c) => c.id === id)?.name ?? "Unknown computer") ?? [];
  const where = names.length > 0 ? names.join(", ") : "Follows your computers";
  const pick = { provider: link?.provider ?? "", model: link?.model ?? "" };
  const model = pick.model && (findModel(providers ?? [], pick)?.name ?? pick.model);
  const startIn = link?.start_in && START_IN_SHORT_LABELS[link.start_in];
  return [where, model, startIn].filter(Boolean).join(" · ");
};
