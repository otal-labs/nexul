import { CalendarIcon, XIcon } from "lucide-react";
import { useState, type KeyboardEvent } from "react";

import { dayKey } from "@nexul/client-core/calendar";

import { TicketCardMenuSub } from "@/components/board/TicketCardMenuSub";
import { Input } from "@/components/ui/input";
import { ContextMenuItem, ContextMenuSeparator, ContextMenuShortcut } from "@/components/ui/context-menu";
import { useSetTicketDueOn } from "@/hooks/TicketHooks";
import type { Ticket } from "@/models/Ticket";
import { formatDateParts, formatDay } from "@/utils/TimeUtility";

interface TicketCardDueMenuProps {
  ticket: Ticket;
}

const QUICK_DAYS = [
  { label: "Today", days: 0 },
  { label: "Tomorrow", days: 1 },
  { label: "In a week", days: 7 },
] as const;

const inDays = (days: number) => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate() + days);
};

// The typed date saves on Enter or when the submenu closes, never per keystroke: typing a year passes through 0002, 0020 and 0202.
export const TicketCardDueMenu = ({ ticket }: TicketCardDueMenuProps) => {
  const setDueOn = useSetTicketDueOn();
  // Only a date typed here saves on close; a quick pick has already saved, and the old draft would undo it.
  const [draft, setDraft] = useState<string | undefined>();
  const save = (dueOn: string) => {
    if (dueOn !== ticket.due_on) setDueOn.mutate({ id: ticket.id, dueOn });
  };
  const onOpenChange = (open: boolean) => {
    if (!open && draft !== undefined) save(draft);
    setDraft(undefined);
  };
  // The menu reads letters as a jump to an item and arrows as moves; inside the field they edit the date. Escape still closes.
  const keepKeys = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") return;
    event.stopPropagation();
    if (event.key !== "Enter" || draft === undefined) return;
    save(draft);
    setDraft(undefined);
  };

  return (
    <TicketCardMenuSub icon={CalendarIcon} label="Due date" value={ticket.due_on ? formatDay(ticket.due_on) : "None"} onOpenChange={onOpenChange}>
      {QUICK_DAYS.map(({ label, days }) => (
        <ContextMenuItem key={label} onSelect={() => save(dayKey(inDays(days)))}>
          {label}
          <ContextMenuShortcut>{formatDateParts(inDays(days), { weekday: "short", day: "numeric" })}</ContextMenuShortcut>
        </ContextMenuItem>
      ))}
      <ContextMenuSeparator />
      <div className="px-1 py-1">
        <Input
          type="date"
          aria-label="Due date"
          value={draft ?? ticket.due_on}
          onChange={(event) => {
            if (event.target.value !== "") setDraft(event.target.value);
          }}
          onKeyDown={keepKeys}
          className="h-8 text-xs"
        />
      </div>
      {ticket.due_on !== "" && (
        <ContextMenuItem onSelect={() => save("")}>
          <XIcon aria-hidden />
          Clear due date
        </ContextMenuItem>
      )}
    </TicketCardMenuSub>
  );
};
