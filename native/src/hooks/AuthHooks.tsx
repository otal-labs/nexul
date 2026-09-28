import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { MeResponse } from "@/models/User";

export const getMeKey = "getMe";

// Refetched on launch and every foreground, so a phone signed out from the web lands on first-run through the client's 401.
export const useFetchMe = (signedIn: boolean) =>
  useQuery({
    queryKey: [getMeKey],
    queryFn: () => api.get<MeResponse>("/api/auth/me"),
    enabled: signedIn,
  });
