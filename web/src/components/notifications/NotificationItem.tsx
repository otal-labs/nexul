import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { formatShortAgo } from "@/components/doc/docTime";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { KeyedTitle } from "@/components/notifications/KeyedTitle";
import { notificationMark, waitingMark } from "@/components/notifications/notificationTone";
import { docQuery } from "@/hooks/DocHooks";
import { useMarkNotificationUnread, useOpenInboxRow, useSelectedInboxRow } from "@/hooks/NotificationHooks";
import { ticketQuery } from "@/hooks/TicketHooks";
import { useHoverPrefetch } from "@/hooks/useHoverPrefetch";
import { NotificationKind, SubjectType } from "@/models/Notification";
import type { InboxRow } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";

interface NotificationItemProps {
  row: InboxRow;
}

// Selecting a row loads it into the detail pane and marks it read — "viewing is reading"; a read row's menu puts it back.
export const NotificationItem = ({ row }: NotificationItemProps) => {
  const selected = useSelectedInboxRow()?.key === row.key;
  const open = useOpenInboxRow();
  const markUnread = useMarkNotificationUnread();
  const unread = row.unreadIds.length > 0;
  // A row still waiting on the viewer leads with the question mark, whatever its newest notice was.
  const { icon: MarkIcon, tone, label } = row.needsAnswer ? waitingMark : notificationMark(row.notification);
  const { subject_type, subject_id, kind } = row.notification;
  const playWorded = row.title === row.notification.subject_title && (kind === NotificationKind.PlayRunFinished || kind === NotificationKind.PlayRunWaiting);
  // The detail pane reads a ticket or a doc by its id; the other subjects load nothing worth fetching early.
  const prefetch = useHoverPrefetch((client) => {
    if (subject_type === SubjectType.Ticket) return client.prefetchQuery(ticketQuery(subject_id));
    if (subject_type === SubjectType.Doc) return client.prefetchQuery(docQuery(subject_id));
    return undefined;
  });
  return (
    <RowContextMenu menu={unread ? undefined : { onMarkUnread: () => markUnread.mutate(row.notification.id) }}>
      <button
        type="button"
        {...prefetch}
        onClick={() => open(row)}
        aria-current={selected}
        className={cn(
          "relative flex w-full items-start px-3 py-2 inset-focus text-left focus-visible:rounded-md text-row leading-5 transition-colors duration-150 ease-standard",
          selected ? currentRowFillClass : "hover:bg-accent",
        )}
      >
        {selected && <CurrentRowEdge />}
        <span className="absolute top-2 left-0 flex h-lh w-3 items-center justify-center">
          <span
            {...(unread ? { role: "img", "aria-label": "Unread" } : { "aria-hidden": true })}
            data-unread={unread || undefined}
            className="unread-dot size-1.5 rounded-full bg-foreground"
          />
        </span>
        <span className="mr-2.5 flex h-lh shrink-0 items-center">
          <MarkIcon
            className={cn("size-4", tone ?? "text-muted-foreground")}
            {...(label ? { role: "img", "aria-label": label } : { "aria-hidden": true })}
          />
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex items-start gap-3">
            <span
              dir="auto"
              title={row.title}
              className={cn("line-clamp-2 min-w-0 flex-1 break-words text-pretty text-foreground", unread && "font-semibold", !unread && selected && "font-medium")}
            >
              <KeyedTitle title={row.title} quiet={playWorded} />
            </span>
            {row.needsAnswer && (
              <span className="inline-flex shrink-0 items-center gap-1.5 text-xs leading-5 text-foreground">
                <span className="size-1.5 rounded-full bg-warning-mark" aria-hidden />
                Needs your answer
              </span>
            )}
            <span className="shrink-0 font-mono text-micro leading-5 text-muted-foreground tabular-nums">{formatShortAgo(row.notification.created_at)}</span>
          </span>
          {row.summary && (
            <span title={row.summary} className="mt-0.5 block truncate text-xs text-muted-foreground">
              {row.summary}
            </span>
          )}
        </span>
      </button>
    </RowContextMenu>
  );
};
