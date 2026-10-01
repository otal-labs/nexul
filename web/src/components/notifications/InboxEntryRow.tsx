import { InboxFolderSection } from "@/components/notifications/InboxFolderSection";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import type { InboxEntry } from "@/utils/InboxUtility";

interface InboxEntryRowProps {
  entry: InboxEntry;
}

export const InboxEntryRow = ({ entry }: InboxEntryRowProps) => (
  <>
    {entry.type === "folder" && <InboxFolderSection folder={entry} />}
    {entry.type !== "folder" && <NotificationItem row={entry} />}
  </>
);
