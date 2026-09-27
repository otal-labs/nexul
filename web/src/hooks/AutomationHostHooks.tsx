import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { AutomationHost, AutomationHostEnrollment, AutomationHostEnrollmentFormData } from "@/models/AutomationHost";

export const getAutomationHostsKey = "automationHosts";

export const useFetchAutomationHosts = () =>
  useQuery({
    queryKey: [getAutomationHostsKey],
    queryFn: async () => (await api.get<AutomationHost[]>("/api/automation-hosts")).data,
  });

export const useCreateAutomationHostEnrollment = () =>
  useMutation({
    mutationFn: async ({ name, machine }: AutomationHostEnrollmentFormData) =>
      (await api.post<AutomationHostEnrollment>("/api/automation-hosts/enrollments", { name, machine: machine || undefined })).data,
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useRemoveAutomationHost = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/api/automation-hosts/${id}`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getAutomationHostsKey] });
      toast.success("Automations host removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
