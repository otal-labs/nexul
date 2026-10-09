import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { referenceDataOptions } from "@/lib/queryClient";
import type { Identity, MeResponse } from "@/models/User";

export const getMeKey = "getMe";
export const getIdentitiesKey = "getIdentities";

// Refetched on launch and every foreground, so a phone signed out from the web lands on first-run through the client's 401.
export const useFetchMe = (signedIn: boolean) =>
  useQuery({
    queryKey: [getMeKey],
    queryFn: () => api.get<MeResponse>("/api/auth/me"),
    enabled: signedIn,
    ...referenceDataOptions,
  });

// Read-only on the phone: Profile shows these, linking/unlinking stays web-only for now.
export const useFetchIdentities = () =>
  useQuery({
    queryKey: [getIdentitiesKey],
    queryFn: () => api.get<{ identities: Identity[] }>("/api/auth/identities"),
  });
