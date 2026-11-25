import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { AuditSettings, AuditSettingsPatch, AuditStreamTest } from "@/ee/models/AuditLog";
import type { LiveFollower } from "@/lib/live";

export const getAuditSettingsKey = "auditSettings";

// Delivery counts change with every row sent, which pushes nothing, so a return to the window reads them again.
export const useAuditSettings = () =>
  useQuery({
    queryKey: [getAuditSettingsKey],
    queryFn: async () => (await api.get<AuditSettings>("/api/instance/audit")).data,
    refetchOnWindowFocus: true,
  });

// new_secret is dropped from the cache, so the secret shows only in the answer of the call that made it.
const keep = (client: QueryClient, data: AuditSettings) => {
  const settings = { ...data };
  delete settings.new_secret;
  client.setQueryData([getAuditSettingsKey], settings);
};

// The refused reason (days out of range, a destination that is not https) comes back for the form's toast.
export const useSaveAuditSettings = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async (patch: AuditSettingsPatch) => (await api.put<AuditSettings>("/api/instance/audit", patch)).data),
    onSuccess: (data) => keep(client, data),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveAuditSettings = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.delete<AuditSettings>("/api/instance/audit")).data),
    onSuccess: (data) => {
      toast.success("Audit log settings removed");
      keep(client, data);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Sends one signed sample to the form's destination; the receiver's answer shows under the field, nothing is saved.
export const useTestAuditStream = () =>
  useMutation({
    mutationFn: requires("instance:write", async (patch: Pick<AuditSettingsPatch, "stream_url" | "stream_secret">) =>
      (await api.post<AuditStreamTest>("/api/instance/audit/stream/test", patch)).data,
    ),
    onError: (error) => toast.error(errorMessage(error)),
  });

export const useRetryAuditStream = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.post<AuditSettings>("/api/instance/audit/stream/retry")).data),
    onSuccess: (data) => {
      toast.success("Failed deliveries queued again");
      keep(client, data);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRotateAuditSecret = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.post<AuditSettings>("/api/instance/audit/stream/rotate-secret")).data),
    onSuccess: (data) => keep(client, data),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const auditLogFollower: LiveFollower = {
  "instance.audit_settings_changed": (_payload: unknown, { client }) => client.invalidateQueries({ queryKey: [getAuditSettingsKey] }),
};
