import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HomeSection } from "@/components/home/HomeSection";
import { InboxListSkeleton } from "@/components/notifications/InboxListSkeleton";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import { useFetchInbox } from "@/hooks/NotificationHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { inboxRows, unreadEntries } from "@/utils/InboxUtility";

const SHOWN = 5;

// The inbox's own rows, so opening one here reads it and lands on it there, as a click in the inbox does.
export const HomeInboxSection = () => {
  const wsPath = useWorkspacePath();
  const { data: entries, error, isPending } = useFetchInbox();
  const unread = entries && inboxRows(unreadEntries(entries, undefined)).sort((a, b) => b.notification.created_at.localeCompare(a.notification.created_at));
  return (
    <HomeSection title="Unread" more={{ to: wsPath("/inbox"), label: "Open inbox" }}>
      {isPending && <InboxListSkeleton />}
      {error && <ErrorDisplay error={error} className="p-6" />}
      {unread && unread.length === 0 && <EmptyRow flush className="py-2">You're all caught up.</EmptyRow>}
      {unread && unread.length > 0 && (
        <div className="-mx-3">
          {unread.slice(0, SHOWN).map((row) => (
            <NotificationItem key={row.key} row={row} />
          ))}
        </div>
      )}
      {unread && unread.length > SHOWN && (
        <p className="pt-2 text-xs text-muted-foreground">
          <span className="font-mono tabular-nums">{unread.length - SHOWN}</span> more in the inbox
        </p>
      )}
    </HomeSection>
  );
};
