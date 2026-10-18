import { useQuery, type QueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import { refetchDeployHistory, type DeployFrame } from "@/hooks/DeployHooks";
import { latestDeploy, type Deploy, type ServiceDef } from "@/models/Service";
import { followEach, type LiveFollower } from "@/lib/live";

export const getServicesKey = "getServices";
export const getServiceDeploysKey = "getServiceDeploys";

// No project lists every project's services, which the workspace-wide topology canvas wires from.
export const useFetchServices = (projectId?: string, enabled = true) =>
  useQuery({
    queryKey: [getServicesKey, projectId ?? "all"],
    queryFn: async () =>
      (await api.get<ServiceDef[]>("/api/services", { params: projectId ? { project_id: projectId } : {} })).data,
    enabled,
  });

// pollMs keeps refetching while a deploy is pending/running, for watchers that cannot rely on catching the live event.
export const useFetchServiceDeploys = (serviceId: string | undefined, pollMs?: number) =>
  useQuery({
    queryKey: [getServiceDeploysKey, serviceId],
    queryFn: async () => (await api.get<Deploy[]>(`/api/services/${serviceId}/deploys`)).data,
    enabled: !!serviceId,
    refetchInterval: (query) => {
      if (!pollMs) return false;
      const latest = latestDeploy(query.state.data);
      return latest?.status === "healthy" || latest?.status === "failed" ? false : pollMs;
    },
  });

// A stack is listed under its project and in the every-project list; one with no project is the instance's own.
const refetchLists = (client: QueryClient, projectId: string) =>
  Promise.all([...new Set([projectId || "all", "all"])].map((list) => client.invalidateQueries({ queryKey: [getServicesKey, list], exact: true })));

export const serviceFollower: LiveFollower = {
  ...followEach(["service.created", "service.updated"], ({ stack }: { stack: { project_id: string } }, { client }) => refetchLists(client, stack.project_id)),
  "service.deleted": ({ project_id }: { project_id?: string }, { client }) => refetchLists(client, project_id ?? ""),
  "deploy.updated": (frame: DeployFrame, { client }) => refetchDeployHistory(client, getServiceDeploysKey, frame),
  // The frame names no project, so a stack's rule, or a machine's for the instance's stacks on it, refetches every services list.
  "access.grant.changed": ({ resource_type }: { resource_type: string }, { client }) =>
    (resource_type === "stack" || resource_type === "machine") && client.invalidateQueries({ queryKey: [getServicesKey] }),
};
