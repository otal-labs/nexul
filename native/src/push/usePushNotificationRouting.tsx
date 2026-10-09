import { useRouter, type Href, type ImperativeRouter } from "expo-router";
import * as Notifications from "expo-notifications";
import { useEffect } from "react";

import { AREA_PERMISSION, projectPermissions } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { subjectRoute } from "@/components/inbox/InboxScreen";
import { getNotificationsKey } from "@/hooks/NotificationHooks";
import { myRoleQuery } from "@/hooks/WorkspaceHooks";
import { queryClient } from "@/lib/queryClient";
import type { Notification } from "@/models/Notification";

// The Inbox caches one list per workspace and a push can come from any of them; the unscoped list holds them all.
const findNotification = async (notificationId: string): Promise<Notification | undefined> => {
  const cached = queryClient
    .getQueriesData<Notification[]>({ queryKey: [getNotificationsKey] })
    .flatMap(([, list]) => list ?? [])
    .find((n) => n.id === notificationId);
  if (cached) return cached;
  const notifications = await api.get<Notification[]>("/api/notifications");
  return notifications.find((n) => n.id === notificationId);
};

// The same answer the Inbox's useAreaAccess gives, for the workspace the notification belongs to.
const canReadTickets = async (workspaceId: string): Promise<boolean> => {
  const role = await queryClient.ensureQueryData(myRoleQuery.options(workspaceId));
  return projectPermissions(role, undefined).includes(AREA_PERMISSION.tickets);
};

// Tapping only navigates; the mark-read call stays where the Inbox already makes it.
const openNotification = async (notificationId: string, router: ImperativeRouter): Promise<void> => {
  router.push("/inbox" as Href);
  const notification = await findNotification(notificationId);
  const target = notification && subjectRoute(notification, await canReadTickets(notification.workspace_id));
  if (target) router.push(target, { withAnchor: true });
};

const notificationIdFrom = (response: Notifications.NotificationResponse): string | undefined => {
  const id = response.notification.request.content.data?.notification_id;
  return typeof id === "string" ? id : undefined;
};

// Handles both a cold start (app opened by the tap) and a warm tap while the app is already running.
export const usePushNotificationRouting = (): void => {
  const router = useRouter();

  useEffect(() => {
    const handle = (response: Notifications.NotificationResponse) => {
      const notificationId = notificationIdFrom(response);
      if (notificationId) void openNotification(notificationId, router);
    };

    const initial = Notifications.getLastNotificationResponse();
    if (initial) {
      handle(initial);
      Notifications.clearLastNotificationResponse();
    }

    const subscription = Notifications.addNotificationResponseReceivedListener(handle);
    return () => subscription.remove();
  }, [router]);
};
