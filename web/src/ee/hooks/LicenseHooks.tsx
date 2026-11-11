import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { License } from "@/ee/models/License";
import type { LiveFollower } from "@/lib/live";

export const getLicenseKey = "license";

export const useLicense = (enabled = true) =>
  useQuery({
    queryKey: [getLicenseKey],
    queryFn: async () => (await api.get<License>("/api/instance/license")).data,
    enabled,
    staleTime: 5 * 60 * 1000,
  });

// Refusals (another instance, a bad signature, expired) come back on the key field for the dialog to show.
export const useSetLicense = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async (key: string) => (await api.put<License>("/api/instance/license", { key })).data),
    onSuccess: (data) => client.setQueryData([getLicenseKey], data),
  });
};

export const useRemoveLicense = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.delete<License>("/api/instance/license")).data),
    onSuccess: (data) => {
      client.setQueryData([getLicenseKey], data);
      toast.success("License key removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const licenseFollower: LiveFollower = {
  "instance.license_changed": (_payload: unknown, { client }) => client.invalidateQueries({ queryKey: [getLicenseKey] }),
};
