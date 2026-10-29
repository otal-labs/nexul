import { CheckCheckIcon, InboxIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneEmpty } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InboxEntryRow } from "@/components/notifications/InboxEntryRow";
import { Button } from "@/components/ui/button";
import { inboxUnreadCount, type InboxEntry } from "@/utils/InboxUtility";

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
}: NotificationsSidebarProps) => {
  const unread = entries ? inboxUnreadCount(entries) : 0;
  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Inbox"
        count={unread}
        hideZeroCount
        controls={
          unread > 0 && (
            <Button variant="ghost" size="icon" className="size-8" onClick={onMarkAllRead} loading={isMarkingAllRead} aria-label="Mark all read" title="Mark all read">
              <CheckCheckIcon aria-hidden />
            </Button>
          )
        }
      />
      <nav aria-label="Notifications" className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-6">
        {isLoading && <LoadingDisplay />}
        {Boolean(error) && <ErrorDisplay error={error} />}
        {entries && entries.length === 0 && <ListPaneEmpty icon={InboxIcon} message="No notifications" />}
        {entries && entries.length > 0 && (
          <EnterList as="div" arrival="rise">
            {entries.map((e) => <InboxEntryRow key={e.key} entry={e} />)}
          </EnterList>
        )}
      </nav>
    </div>
  );
};
