import { useState } from "react";

import { NotificationDetailPanel } from "@/components/notifications/NotificationDetailPanel";
import { NotificationsSidebar } from "@/components/notifications/NotificationsSidebar";
import {
  useFetchNotifications,
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
} from "@/hooks/NotificationHooks";
import type { Notification } from "@/models/Notification";

export const InboxPage = () => {
  const { data, error, isPending } = useFetchNotifications();
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const selected = data?.find((n) => n.id === selectedId) ?? data?.[0] ?? null;

  const onSelect = (notification: Notification) => {
    setSelectedId(notification.id);
    if (!notification.read) markRead.mutate(notification.id);
  };

  return (
    <div className="flex h-screen">
      <NotificationsSidebar
        notifications={data}
        error={error}
        isLoading={isPending}
        selectedId={selected?.id}
        onSelect={onSelect}
        onMarkAllRead={() => markAllRead.mutate()}
        isMarkingAllRead={markAllRead.isPending}
      />
      <NotificationDetailPanel selected={selected} isLoading={isPending} />
    </div>
  );
};
