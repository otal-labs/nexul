import { NotificationDetailPanel } from "@/components/notifications/NotificationDetailPanel";
import { NotificationsSidebar } from "@/components/notifications/NotificationsSidebar";
import { useFetchInbox, useMarkAllNotificationsRead, useSelectedInboxRow } from "@/hooks/NotificationHooks";

export const InboxPage = () => {
  const { data, error, isPending } = useFetchInbox();
  const selected = useSelectedInboxRow();
  const markAllRead = useMarkAllNotificationsRead();

  return (
    <div data-pane-layout className="flex h-full gap-2">
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
