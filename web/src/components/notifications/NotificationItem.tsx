import { CircleIcon } from "lucide-react";

import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { notificationKindIcon } from "@/components/notifications/notificationKindIcon";
import { notificationToneText } from "@/components/notifications/notificationTone";
import { useMarkNotificationUnread, useOpenInboxRow, useSelectedInboxRow } from "@/hooks/NotificationHooks";
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
  const KindIcon = notificationKindIcon[row.notification.kind] ?? CircleIcon;
  return (
    <RowContextMenu menu={unread ? undefined : { onMarkUnread: () => markUnread.mutate(row.notification.id) }}>
      <button
        type="button"
        onClick={() => open(row)}
        aria-current={selected}
        className={cn(
          "relative flex w-full items-start px-3 py-2 inset-focus text-left focus-visible:rounded-md text-[13px] leading-5 transition-colors duration-150 ease-standard",
          selected ? currentRowFillClass : "hover:bg-accent/40",
        )}
      >
        {selected && <CurrentRowEdge />}
        <span className="flex h-lh w-3 shrink-0 items-center">
          <span
            {...(unread ? { role: "img", "aria-label": "Unread" } : { "aria-hidden": true })}
            data-unread={unread || undefined}
            className="unread-dot size-1.5 rounded-full bg-foreground"
          />
        </span>
        <span className="min-w-0 flex-1">
          <span
            dir="auto"
            title={row.title}
            className={cn("line-clamp-2 break-words text-pretty text-foreground", unread && "font-semibold", !unread && selected && "font-medium")}
          >
            {row.title}
          </span>
          <span className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
            <KindIcon className={cn("size-3.5 shrink-0", notificationToneText[row.notification.kind])} aria-hidden />
            <span title={row.summary} className="min-w-0 flex-1 truncate">{row.summary}</span>
            <span className="shrink-0 font-mono text-[11px] tabular-nums">{formatUpdatedAgo(row.notification.created_at)}</span>
          </span>
        </span>
      </button>
    </RowContextMenu>
  );
};
