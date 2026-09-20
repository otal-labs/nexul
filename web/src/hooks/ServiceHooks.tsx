import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { latestDeploy, type Deploy, type ServiceDef } from "@/models/Service";

export const getServicesKey = "getServices";
const getServiceKey = "getService";
export const getServiceDeploysKey = "getServiceDeploys";

// No project lists every project's services, which the workspace-wide topology canvas wires from.
export const useFetchServices = (projectId?: string) =>
  useQuery({
    queryKey: [getServicesKey, projectId ?? "all"],
    queryFn: async () =>
      (await api.get<ServiceDef[]>("/api/services", { params: projectId ? { project_id: projectId } : {} })).data,
  });

export const useFetchService = (id: string | undefined) =>
  useQuery({
    queryKey: [getServiceKey, id],
    queryFn: async () => (await api.get<ServiceDef>(`/api/services/${id}`)).data,
    enabled: !!id,
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

export const useCreateService = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (payload: Partial<ServiceDef>) =>
      (await api.post<ServiceDef>("/api/services", payload)).data,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Service created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useUpdateService = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: Partial<ServiceDef> }) =>
      (await api.patch<ServiceDef>(`/api/services/${id}`, payload)).data,
    onSuccess: async (_data, vars) => {
      await client.invalidateQueries({ queryKey: [getServiceKey, vars.id] });
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Service updated");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteService = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/services/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getServicesKey] });
      toast.success("Service removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeployService = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ serviceId, image, ref }: { serviceId: string; image?: string; ref?: string }) =>
      (await api.post<Deploy>("/api/deploys", { service_id: serviceId, image, ref })).data,
    onSuccess: async (_data, vars) => {
      await client.invalidateQueries({ queryKey: [getServiceDeploysKey, vars.serviceId] });
      toast.success("Deploy triggered");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRollbackService = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (serviceId: string) =>
      (await api.post<Deploy>(`/api/services/${serviceId}/rollback`)).data,
    onSuccess: async (_data, serviceId) => {
      await client.invalidateQueries({ queryKey: [getServiceDeploysKey, serviceId] });
      toast.success("Rollback triggered");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
