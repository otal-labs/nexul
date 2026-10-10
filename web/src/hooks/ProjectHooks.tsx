import { type QueryClient, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { RepoRole, TestsLocation } from "@/enums/Project";
import type { DeleteImpact, Project, ProjectAccessEntry, ProjectSetup, RepoRef, SetupMark } from "@/models/Project";
import type { Repo } from "@/models/Repository";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { refetchHolding, type LiveFollower } from "@/lib/live";

export const getProjectsKey = "getProjects";
export const getProjectKey = "getProject";
export const getProjectAccessKey = "getProjectAccess";
const getProjectReposKey = "getProjectRepos";

// The projects of a workspace the viewer may open; Team and invitations read workspaces other than the selected one.
export const useFetchWorkspaceProjects = (workspaceId: string, enabled = true) =>
  useQuery({
    queryKey: [getProjectsKey, workspaceId],
    queryFn: async () => (await api.get<Project[]>("/api/projects", { params: { workspace_id: workspaceId } })).data,
    enabled: enabled && !!workspaceId,
  });

// Reads selectedWorkspaceId internally so the key refetches on workspace switch with no per-caller wiring.
export const useFetchProjects = (enabled = true) =>
  useFetchWorkspaceProjects(useWorkspaceStore((s) => s.selectedWorkspaceId), enabled);

// The Restricted members who may open a project; the server answers holders of members:write only.
export const useFetchProjectAccess = (projectId: string, enabled = true) =>
  useQuery({
    queryKey: [getProjectAccessKey, projectId],
    queryFn: async () => (await api.get<{ access: ProjectAccessEntry[] }>(`/api/projects/${projectId}/access`)).data.access,
    enabled,
  });

export const useFetchProject = (id: string | undefined) =>
  useQuery({
    queryKey: [getProjectKey, id],
    queryFn: async () => (await api.get<Project>(`/api/projects/${id}`)).data,
    enabled: !!id,
  });

export const useFetchProjectDeleteImpact = (id: string | undefined) =>
  useQuery({
    queryKey: ["getProjectImpact", id],
    queryFn: async () => (await api.get<DeleteImpact>(`/api/projects/${id}/impact`)).data,
    enabled: !!id,
  });

export const useCreateProject = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ name, prefix, icon }: { name: string; prefix: string; icon?: string }) =>
      (
        await api.post<Project>("/api/projects", {
          name,
          prefix,
          icon,
          workspace_id: useWorkspaceStore.getState().selectedWorkspaceId,
          setup_finished: false,
        })
      ).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectsKey] });
      toast.success("Project created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface SetupChangedPayload {
  project_id: string;
  workspace_id: string;
  setup: ProjectSetup;
}

// Writes a project's setup into the list the sidebar reads and the project's own read, so its nav flips in place.
const writeSetup = (client: QueryClient, { project_id, workspace_id }: SetupChangedPayload, next: (current: ProjectSetup) => ProjectSetup) => {
  client.setQueryData<Project[]>([getProjectsKey, workspace_id], (list) =>
    list?.map((p) => (p.id === project_id ? { ...p, setup: next(p.setup) } : p)),
  );
  client.setQueryData<Project>([getProjectKey, project_id], (project) => project && { ...project, setup: next(project.setup) });
};

// A frame has one audience, so it carries the finished flag and the marks but not the service, which needs stacks:read:
// those patch in at once and the reads that hold the project refetch for the rest (ADR 0143).
const followSetup = (client: QueryClient, payload: SetupChangedPayload) => {
  writeSetup(client, payload, (current) => ({ ...current, finished: payload.setup.finished, steps: payload.setup.steps }));
  return Promise.all([
    client.invalidateQueries({ queryKey: [getProjectKey, payload.project_id], exact: true }),
    refetchHolding(client, [getProjectsKey, payload.workspace_id], (list: Project[]) => list.some((p) => p.id === payload.project_id)),
  ]);
};

export interface SetupChangeInput {
  projectId: string;
  finished?: boolean;
  stack_id?: string;
  env_keys?: string[];
  steps?: Partial<Record<string, SetupMark>>;
}

export const useChangeProjectSetup = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ projectId, ...change }: SetupChangeInput) =>
      (await api.put<Project>(`/api/projects/${projectId}/setup`, change)).data,
    onSuccess: (project) =>
      writeSetup(client, { project_id: project.id, workspace_id: project.workspace_id, setup: project.setup }, () => project.setup),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRenameProject = () => {
  const client = useQueryClient();
  return useMutation({
    // icon omitted keeps the current one ("no change" server-side); "" explicitly clears it.
    mutationFn: async ({ id, name, icon }: { id: string; name: string; icon?: string }) =>
      (await api.patch<Project>(`/api/projects/${id}`, icon === undefined ? { name } : { name, icon })).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectsKey] });
      toast.success("Project renamed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteProject = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/projects/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectsKey] });
      toast.success("Project removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useFetchProjectRepos = (projectId: string | undefined) =>
  useQuery({
    queryKey: [getProjectReposKey, projectId],
    queryFn: async () => (await api.get<RepoRef[]>(`/api/projects/${projectId}/repos`)).data,
    enabled: !!projectId,
  });

export const useAddProjectRepo = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({
      projectId,
      owner,
      name,
      connectorId,
    }: {
      projectId: string;
      owner: string;
      name: string;
      connectorId: string;
    }) => api.post(`/api/projects/${projectId}/repos`, { owner, name, connector_id: connectorId }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectReposKey] });
      toast.success("Repository added");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface SaveTestsAnswerInput {
  projectId: string;
  testsLocation: TestsLocation;
  testsRepo: Repo | null;
  attached: RepoRef[];
}

// Attaching a tests repository records "separate" server-side; every other answer is recorded as it stands.
export const useSaveTestsAnswer = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ projectId, testsLocation, testsRepo, attached }: SaveTestsAnswerInput) => {
      const attach =
        testsLocation === TestsLocation.Separate &&
        testsRepo &&
        !attached.some((r) => r.owner === testsRepo.owner && r.name === testsRepo.name && r.role === RepoRole.Tests);
      if (attach) {
        await api.post(`/api/projects/${projectId}/repos`, {
          owner: testsRepo.owner,
          name: testsRepo.name,
          connector_id: "github",
          role: RepoRole.Tests,
        });
        return;
      }
      await api.put(`/api/projects/${projectId}/tests-location`, { tests_location: testsLocation });
    },
    onSuccess: async (_, { projectId }) => {
      await client.invalidateQueries({ queryKey: [getProjectReposKey, projectId] });
      await client.invalidateQueries({ queryKey: [getProjectKey, projectId] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveProjectRepo = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ owner, name }: { owner: string; name: string }) =>
      api.delete(`/api/projects/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getProjectReposKey] });
      toast.success("Repository removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Who a manager sees with access to a project follows someone else's Project access moving.
export const projectFollower: LiveFollower = {
  "project.setup_changed": (payload: SetupChangedPayload, { client }) => followSetup(client, payload),
  "workspace.member.updated": ({ project_ids }: { project_ids?: string[] }, { client }) =>
    Promise.all((project_ids ?? []).map((id) => client.invalidateQueries({ queryKey: [getProjectAccessKey, id], exact: true }))),
  "access.grant.changed": ({ resource_type, resource_id }: { resource_type: string; resource_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: resource_type === "project" ? [getProjectAccessKey, resource_id] : [getProjectAccessKey] }),
};
