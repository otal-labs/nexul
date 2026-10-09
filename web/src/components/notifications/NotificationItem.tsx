import { notificationKindIcon } from "@/components/notifications/notificationKindIcon";
import { useOpenInboxRow, useSelectedInboxRow } from "@/hooks/NotificationHooks";
import type { InboxRow } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";
import { CircleIcon } from "lucide-react";

interface NotificationItemProps {
  row: InboxRow;
  /** Set for a row inside a folder group, so its title lines up under the folder's name. */
  inset?: boolean;
}

// Selecting a row loads it into the detail pane and marks it read — "viewing is reading", no separate control.
export const NotificationItem = ({ row, inset = false }: NotificationItemProps) => {
  const selected = useSelectedInboxRow()?.key === row.key;
  const open = useOpenInboxRow();
  const unread = row.unreadIds.length > 0;
  const KindIcon = notificationKindIcon[row.notification.kind] ?? CircleIcon;
  return (
    <button
      type="button"
      onClick={() => open(row)}
      aria-current={selected}
      className={cn(
        "flex w-full items-start gap-2.5 border-b border-l-2 border-b-border px-4 py-3 text-left transition-colors duration-150 ease-standard",
        inset && "pl-9",
        selected ? "border-l-brand bg-accent" : "border-l-transparent hover:bg-accent/40",
      )}
    >
      <span className="relative mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md bg-muted/60 text-muted-foreground">
        <KindIcon className="size-3.5" aria-hidden />
        <span
          {...(unread ? { role: "img", "aria-label": "Unread" } : { "aria-hidden": true })}
          data-unread={unread || undefined}
          className="unread-dot absolute -top-0.5 -right-0.5 size-2 rounded-full bg-foreground ring-2 ring-panel"
        />
      </span>
      <div className="min-w-0 flex-1">
        <p className="flex items-baseline gap-2">
          <span
            dir="auto"
            title={row.title}
            className={cn("line-clamp-2 min-w-0 flex-1 text-[13px] break-words", unread ? "font-semibold" : "font-medium text-muted-foreground")}
          >
            {row.title}
          </span>
          <span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums">{formatRelativeTime(row.notification.created_at)}</span>
        </p>
        <p className="mt-0.5 truncate text-xs text-muted-foreground">{row.summary}</p>
      </div>
    </button>
  );
};
