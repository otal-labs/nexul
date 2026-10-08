import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { InboxEntryRow } from "@/components/notifications/InboxEntryRow";
import { Button } from "@/components/ui/button";
import type { InboxEntry } from "@/utils/InboxUtility";

interface NotificationsSidebarProps {
  entries: InboxEntry[] | undefined;
  error: unknown;
  isLoading: boolean;
  onMarkAllRead: () => void;
  isMarkingAllRead: boolean;
}

export const NotificationsSidebar = ({
  entries,
  error,
  isLoading,
  onMarkAllRead,
  isMarkingAllRead,
}: NotificationsSidebarProps) => (
  <div className="flex w-80 shrink-0 flex-col border-r border-border">
    <div className="flex h-14 items-center justify-between border-b border-border px-4">
      <h1 className="text-sm font-semibold">Inbox</h1>
      {entries && entries.length > 0 && (
        <Button variant="ghost" size="sm" onClick={onMarkAllRead} loading={isMarkingAllRead}>
          Mark all read
        </Button>
      )}
    </div>
    <nav aria-label="Notifications" className="flex-1 overflow-y-auto">
      {isLoading && <LoadingDisplay />}
      {Boolean(error) && <ErrorDisplay error={error} />}
      {entries && entries.length === 0 && <NoDataDisplay message="No notifications" />}
      {entries?.map((e) => <InboxEntryRow key={e.key} entry={e} />)}
    </nav>
  </div>
);
