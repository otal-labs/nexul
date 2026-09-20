import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Notification, UnreadCount } from "@/models/Notification";

export const getNotificationsKey = "getNotifications";
export const getUnreadCountKey = "getUnreadCount";

export const useFetchNotifications = () =>
  useQuery({
    queryKey: [getNotificationsKey],
    queryFn: async () => (await api.get<Notification[]>("/api/notifications")).data,
  });

export const useFetchUnreadCount = (enabled = true) =>
  useQuery({
    queryKey: [getUnreadCountKey],
    queryFn: async () => (await api.get<UnreadCount>("/api/notifications/unread-count")).data,
    enabled,
  });

export const useMarkNotificationRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.post(`/api/notifications/${id}/read`),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useMarkAllNotificationsRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => api.post("/api/notifications/read-all"),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [getNotificationsKey] });
      await client.invalidateQueries({ queryKey: [getUnreadCountKey] });
      toast.success("All notifications marked read");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
