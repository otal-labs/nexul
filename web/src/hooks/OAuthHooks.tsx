import { useMutation, useQuery } from "@tanstack/react-query";

import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { getOAuthConsentKey, type OAuthConsent, type OAuthDecision } from "@/models/OAuth";

// search is the authorize request's own query string, passed on untouched so the server checks exactly what the app sent.
export const useFetchOAuthConsent = (search: string) =>
  useQuery({
    queryKey: [getOAuthConsentKey, search],
    queryFn: async () => (await api.get<OAuthConsent>(`/api/auth/oauth/authorize${search}`)).data,
    retry: false,
  });

export const useDecideOAuth = (search: string) =>
  useMutation({
    mutationFn: requires("none", async (approve: boolean) =>
      (await api.post<OAuthDecision>("/api/auth/oauth/authorize", { ...Object.fromEntries(new URLSearchParams(search)), approve })).data),
  });
