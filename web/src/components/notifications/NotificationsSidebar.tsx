import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import { Button } from "@/components/ui/button";
import type { Notification } from "@/models/Notification";

interface NotificationsSidebarProps {
  notifications: Notification[] | undefined;
  error: unknown;
  isLoading: boolean;
  selectedId: string | undefined;
  onSelect: (notification: Notification) => void;
  onMarkAllRead: () => void;
  isMarkingAllRead: boolean;
}

export const NotificationsSidebar = ({
  notifications,
  error,
  isLoading,
  selectedId,
  onSelect,
  onMarkAllRead,
  isMarkingAllRead,
}: NotificationsSidebarProps) => (
  <div className="flex w-80 shrink-0 flex-col border-r border-border">
    <div className="flex h-14 items-center justify-between border-b border-border px-4">
      <h1 className="text-sm font-semibold">Inbox</h1>
      {notifications && notifications.length > 0 && (
        <Button variant="ghost" size="sm" onClick={onMarkAllRead} disabled={isMarkingAllRead}>
          Mark all read
        </Button>
      )}
    </div>
    <nav aria-label="Notifications" className="flex-1 overflow-y-auto">
      {isLoading && <LoadingDisplay />}
      {Boolean(error) && <ErrorDisplay error={error} />}
      {notifications && notifications.length === 0 && <NoDataDisplay message="No notifications" />}
      {notifications?.map((n) => (
        <NotificationItem key={n.id} notification={n} selected={n.id === selectedId} onSelect={() => onSelect(n)} />
      ))}
    </nav>
  </div>
);
