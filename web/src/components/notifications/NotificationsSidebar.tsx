import { CheckCheckIcon, InboxIcon } from "lucide-react";
import { useRef } from "react";
import { useParams } from "react-router";

import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneEmpty } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InboxEntryRow } from "@/components/notifications/InboxEntryRow";
import { InboxFilterToggle } from "@/components/notifications/InboxFilterToggle";
import { Button } from "@/components/ui/button";
import { useInboxFilterStore } from "@/stores/inboxFilterStore";
import { inboxUnreadCount, unreadEntries, type InboxEntry } from "@/utils/InboxUtility";

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
  const { rowKey } = useParams();
  const unreadOnly = useInboxFilterStore((s) => s.filter === "unread");
  const unread = entries ? inboxUnreadCount(entries) : 0;
  const shown = entries && unreadOnly ? unreadEntries(entries, rowKey) : entries;
  const listRef = useRef<HTMLElement>(null);
  // The button leaves once nothing is unread, so focus moves to the list's first row instead of dropping to the page.
  const markAllRead = () => {
    listRef.current?.querySelector<HTMLElement>("button")?.focus();
    onMarkAllRead();
  };
  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Inbox"
        count={unread}
        hideZeroCount
        controls={
          <>
            {unread > 0 && (
              <Button variant="ghost" size="icon" className="size-8" onClick={markAllRead} loading={isMarkingAllRead} aria-label="Mark all read" title="Mark all read">
                <CheckCheckIcon aria-hidden />
              </Button>
            )}
            <InboxFilterToggle />
          </>
        }
      />
      <nav ref={listRef} aria-label="Notifications" className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-6">
        {isLoading && <LoadingDisplay />}
        {Boolean(error) && <ErrorDisplay error={error} />}
        {entries && entries.length === 0 && <ListPaneEmpty icon={InboxIcon} message="No notifications" hint="Assignments, mentions and changes to what you follow land here." />}
        {entries && entries.length > 0 && shown && shown.length === 0 && <ListPaneEmpty icon={InboxIcon} message="Nothing unread" hint="All shows what you've read." />}
        {shown && shown.length > 0 && (
          <EnterList as="div" arrival="rise">
            {shown.map((e) => <InboxEntryRow key={e.key} entry={e} />)}
          </EnterList>
        )}
      </nav>
    </div>
  );
};
