import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { Notification, UnreadCount } from "@/models/Notification";

export const getNotificationsKey = "getNotifications";
export const getUnreadCountKey = "getUnreadCount";

export const useFetchNotifications = () =>
  useQuery({
    queryKey: [getNotificationsKey],
    queryFn: () => api.get<Notification[]>("/api/notifications"),
  });

export const useFetchUnreadCount = (enabled = true) =>
  useQuery({
    queryKey: [getUnreadCountKey],
    queryFn: () => api.get<UnreadCount>("/api/notifications/unread-count"),
    enabled,
  });

export const useMarkNotificationRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post(`/api/notifications/${id}/read`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
  });
};

export const useMarkAllNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => api.post("/api/notifications/read-all"),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
  });
};

// Hides the tab badge once nothing is unread, rather than showing a stray "0".
export const unreadBadge = (count: number | undefined): number | undefined =>
  count && count > 0 ? count : undefined;
