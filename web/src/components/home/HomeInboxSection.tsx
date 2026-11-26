import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HomeSection } from "@/components/home/HomeSection";
import { InboxListSkeleton } from "@/components/notifications/InboxListSkeleton";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import { useFetchInbox } from "@/hooks/NotificationHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { HomeMoreLink } from "@/components/home/HomeMoreLink";
import { inboxRows } from "@/utils/InboxUtility";

const SHOWN = 6;

// The inbox's own rows, unread first, so a quiet inbox still shows what happened lately; opening one reads it, as in the inbox.
export const HomeInboxSection = () => {
  const wsPath = useWorkspacePath();
  const { data: entries, error, isPending } = useFetchInbox();
  const rows =
    entries &&
    inboxRows(entries).sort(
      (a, b) => Number(b.unreadIds.length > 0) - Number(a.unreadIds.length > 0) || b.notification.created_at.localeCompare(a.notification.created_at),
    );
  return (
    <HomeSection title="Inbox" more={{ to: wsPath("/inbox"), label: "Open inbox" }}>
      {isPending && <InboxListSkeleton />}
      {error && <ErrorDisplay error={error} className="p-6" />}
      {rows && rows.length === 0 && <EmptyRow flush className="py-2">Nothing has reached your inbox yet.</EmptyRow>}
      {rows && rows.length > 0 && (
        <div className="-mx-3">
          {rows.slice(0, SHOWN).map((row) => (
            <NotificationItem key={row.key} row={row} />
          ))}
        </div>
      )}
      {rows && rows.length > SHOWN && <HomeMoreLink to={wsPath("/inbox")} count={rows.length - SHOWN} label="more in the inbox" />}
    </HomeSection>
  );
};
