import { useParams } from "react-router";

import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { NotificationDetailPanel } from "@/components/notifications/NotificationDetailPanel";
import { NotificationsSidebar } from "@/components/notifications/NotificationsSidebar";
import { useFetchInbox, useMarkAllNotificationsRead, useSelectedInboxRow } from "@/hooks/NotificationHooks";
import { EmbeddedCrumbsContext } from "@/hooks/useEmbeddedCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

// The open row is the path's; a row that is no longer listed falls back to the list.
export const InboxPage = () => {
  const { rowKey } = useParams();
  const { data, error, isPending } = useFetchInbox();
  const selected = useSelectedInboxRow();
  const markAllRead = useMarkAllNotificationsRead();
  const wsPath = useWorkspacePath();

  return (
    <EmbeddedCrumbsContext value={[{ label: "Inbox", to: wsPath("/inbox") }]}>
      <ListDetailLayout
        hasSelection={!!rowKey && (isPending || !!selected)}
        list={
          <NotificationsSidebar
            entries={data}
            error={error}
            isLoading={isPending}
            onMarkAllRead={() => markAllRead.mutate()}
            isMarkingAllRead={markAllRead.isPending}
          />
        }
        placeholder="Select a notification to view it"
        detail={<NotificationDetailPanel selected={selected?.notification} />}
      />
    </EmbeddedCrumbsContext>
  );
};
