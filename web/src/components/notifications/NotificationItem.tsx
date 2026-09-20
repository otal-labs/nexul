import { NotificationKind, type Notification } from "@/models/Notification";
import { cn } from "@/lib/utils";

const kindLabels: Record<NotificationKind, string> = {
  [NotificationKind.TicketAssigned]: "assigned to you",
  [NotificationKind.TicketMentioned]: "mentioned you",
  [NotificationKind.TicketStatusChanged]: "status changed",
  [NotificationKind.DocCreated]: "doc created",
  [NotificationKind.DocUpdated]: "doc updated",
  [NotificationKind.MemoryUpdated]: "memory updated",
  [NotificationKind.PlayRunFinished]: "play run ended",
  [NotificationKind.PlayRunWaiting]: "needs your answer",
};

interface NotificationItemProps {
  notification: Notification;
  selected: boolean;
  onSelect: () => void;
}

// Selecting a row loads it into the detail pane and marks it read — "viewing is reading", no separate control.
export const NotificationItem = ({ notification: n, selected, onSelect }: NotificationItemProps) => (
  <button
    type="button"
    onClick={onSelect}
    aria-current={selected}
    className={cn(
      "flex w-full items-start gap-2.5 border-b border-border px-4 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/60",
      selected && "bg-accent",
    )}
  >
    {!n.read && <span className="mt-1.5 size-2 shrink-0 rounded-full bg-primary" aria-label="Unread" />}
    <div className={cn("min-w-0 flex-1", n.read && "pl-[14px]")}>
      <p className={cn("truncate text-sm", !n.read ? "font-semibold" : "font-medium text-muted-foreground")}>
        {n.subject_title}
      </p>
      <p className="truncate text-xs text-muted-foreground">{kindLabels[n.kind]}</p>
    </div>
  </button>
);
