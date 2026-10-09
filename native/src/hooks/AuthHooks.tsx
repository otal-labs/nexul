import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import type { Identity, MeResponse } from "@/models/User";

export const getMeKey = "getMe";
export const getIdentitiesKey = "getIdentities";

// Refetched on launch and every foreground, so a phone signed out from the web lands on first-run through the client's 401.
const meQuery = defineQuery({
  key: getMeKey,
  fetch: () => api.get<MeResponse>("/api/auth/me"),
  refreshes: { "account.profile_updated": "all" },
  untilPushed: true,
});

export const useFetchMe = (signedIn: boolean) => useQuery({ ...meQuery.options(), enabled: signedIn });

const identitiesQuery = defineQuery({
  key: getIdentitiesKey,
  fetch: () => api.get<{ identities: Identity[] }>("/api/auth/identities"),
  refreshes: {},
});

// Read-only on the phone: Profile shows these, linking/unlinking stays web-only for now.
export const useFetchIdentities = () => useQuery(identitiesQuery.options());
