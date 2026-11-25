import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { AuditSettings } from "@/ee/models/AuditLog";
import type { LiveFollower } from "@/lib/live";

export const getAuditSettingsKey = "auditSettings";

export const useAuditSettings = () =>
  useQuery({
    queryKey: [getAuditSettingsKey],
    queryFn: async () => (await api.get<AuditSettings>("/api/instance/audit")).data,
  });

export const auditLogFollower: LiveFollower = {
  "instance.audit_settings_changed": (_payload: unknown, { client }) => client.invalidateQueries({ queryKey: [getAuditSettingsKey] }),
};
