import { useOpenInboxRow, useSelectedInboxRow } from "@/hooks/NotificationHooks";
import type { InboxRow } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

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
  return (
    <button
      type="button"
      onClick={() => open(row)}
      aria-current={selected}
      className={cn(
        "flex w-full items-start gap-2.5 border-b border-border px-4 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/60",
        inset && "pl-9",
        selected && "bg-accent",
      )}
    >
      <span className={cn("mt-1.5 size-2 shrink-0 rounded-full", unread && "bg-primary")} aria-label={unread ? "Unread" : undefined} />
      <div className="min-w-0 flex-1">
        <p className={cn("truncate text-sm", unread ? "font-semibold" : "font-medium text-muted-foreground")}>{row.title}</p>
        <p className="flex gap-2 text-xs text-muted-foreground">
          <span className="truncate">{row.summary}</span>
          {row.type === "doc" && (
            <span className="ml-auto shrink-0 font-mono tabular-nums">{formatRelativeTime(row.notification.created_at)}</span>
          )}
        </p>
      </div>
    </button>
  );
};
