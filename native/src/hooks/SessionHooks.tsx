import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Session } from "@/models/User";
import { clearPushToken } from "@/push/pushToken";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

export const getSessionsKey = "getSessions";

export const useListSessions = () =>
  useQuery({
    queryKey: [getSessionsKey],
    queryFn: () => api.get<{ sessions: Session[] }>("/api/auth/sessions"),
  });

export const useSignOutSession = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(`/api/auth/sessions/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSessionsKey] }),
  });
};

export const useSignOutOtherSessions = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => api.delete("/api/auth/sessions/others"),
    onSuccess: () => client.invalidateQueries({ queryKey: [getSessionsKey] }),
  });
};

// Clears the push token while the session can still authenticate, then deletes the session; the local sign-out always follows.
export const useSignOut = () =>
  useMutation({
    mutationFn: async () => {
      const { host } = useSessionStore.getState();
      const token = readSessionToken();
      if (host && token) await clearPushToken(host, token);
      return api.delete("/api/auth/sessions/current");
    },
    onSettled: () => useSessionStore.getState().signOut(),
  });
