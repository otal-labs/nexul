import { NotificationDetailPanel } from "@/components/notifications/NotificationDetailPanel";
import { NotificationsSidebar } from "@/components/notifications/NotificationsSidebar";
import { useFetchInbox, useMarkAllNotificationsRead, useSelectedInboxRow } from "@/hooks/NotificationHooks";

export const InboxPage = () => {
  const { data, error, isPending } = useFetchInbox();
  const selected = useSelectedInboxRow();
  const markAllRead = useMarkAllNotificationsRead();

  return (
    <div className="flex h-screen">
      <NotificationsSidebar
        entries={data}
        error={error}
        isLoading={isPending}
        onMarkAllRead={() => markAllRead.mutate()}
        isMarkingAllRead={markAllRead.isPending}
      />
      <NotificationDetailPanel selected={selected?.notification ?? null} isLoading={isPending} />
    </div>
  );
};
