import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import type { Branding, Look } from "@/ee/models/Branding";
import type { LiveFollower } from "@/lib/live";

export const getLookKey = "branding-look";
export const getBrandingKey = "branding";

// The server writes the look into the page it serves, so the first paint and the first render already wear it.
const injectedLook = (): Look | undefined => {
  try {
    const text = document.getElementById("branding-data")?.textContent;
    return text ? (JSON.parse(text) as Look) : undefined;
  } catch {
    return undefined;
  }
};

// Public: the sign-in page wears the brand before anyone has signed in.
export const useLook = () =>
  useQuery({
    queryKey: [getLookKey],
    queryFn: async () => (await api.get<Look>("/api/branding")).data,
    initialData: injectedLook,
    staleTime: 5 * 60 * 1000,
  });

export const useBranding = () =>
  useQuery({
    queryKey: [getBrandingKey],
    queryFn: async () => (await api.get<Branding>("/api/instance/branding")).data,
    staleTime: 5 * 60 * 1000,
  });

export interface BrandingPatch {
  product_name?: string;
  accent?: string;
  theme_file?: string;
  remove_theme?: "light" | "dark" | "all";
  css?: string;
}

// A save changes what every page paints, so both reads refresh with it.
const useRefreshBranding = () => {
  const client = useQueryClient();
  return (data: Branding) => {
    client.setQueryData([getBrandingKey], data);
    return client.invalidateQueries({ queryKey: [getLookKey] });
  };
};

export const useUpdateBranding = () => {
  const refresh = useRefreshBranding();
  return useMutation({
    mutationFn: requires("instance:write", async (patch: BrandingPatch) => (await api.patch<Branding>("/api/instance/branding", patch)).data),
    onSuccess: refresh,
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useSetBrandingLogo = () => {
  const refresh = useRefreshBranding();
  return useMutation({
    mutationFn: requires("instance:write", async (file: File) => (await api.put<Branding>("/api/instance/branding/logo", file, { headers: { "Content-Type": file.type } })).data),
    onSuccess: refresh,
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useRemoveBrandingLogo = () => {
  const refresh = useRefreshBranding();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.delete<Branding>("/api/instance/branding/logo")).data),
    onSuccess: refresh,
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useClearBranding = () => {
  const refresh = useRefreshBranding();
  return useMutation({
    mutationFn: requires("instance:write", async () => (await api.delete<Branding>("/api/instance/branding")).data),
    onSuccess: async (data) => {
      await refresh(data);
      toast.success("Branding removed");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Every open page follows a change at once: the look is what each person's own pages paint.
export const brandingFollower: LiveFollower = {
  "instance.branding_changed": (_payload: unknown, { client }) =>
    Promise.all([client.invalidateQueries({ queryKey: [getLookKey] }), client.invalidateQueries({ queryKey: [getBrandingKey] })]),
};
