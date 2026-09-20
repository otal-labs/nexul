import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ticketTypeColor } from "@/components/board/ticketTypeColor";
import { ticketTypeIcon } from "@/components/board/ticketTypeIcon";
import { editableRowClass, menuItemClass, rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/models/Ticket";

interface TicketTypeRowProps {
  ticket: Ticket;
  onSetType?: (ticketId: string, typeId: string) => Promise<void> | void;
}

export const TicketTypeRow = ({ ticket, onSetType }: TicketTypeRowProps) => {
  const { data: ticketTypes } = useFetchProjectTicketTypes(ticket.project_id);
  const types = ticketTypes ?? [];
  const current = types.find((t) => t.id === ticket.type_id);
  const Icon = ticketTypeIcon(current?.name ?? "");

  const content = (
    <>
      <Icon className={cn("size-3.5 shrink-0", ticketTypeColor(current?.name ?? ""))} aria-hidden />
      <span className={cn("text-xs", current ? "font-medium text-foreground" : "text-muted-foreground")}>
        {current?.name ?? "No type"}
      </span>
    </>
  );

  if (!onSetType) {
    return (
      <div className={rowClass}>
        <span className="sr-only">Type</span>
        {content}
      </div>
    );
  }

  return (
    <>
      <span className="sr-only">Type</span>
      <Popover>
        <PopoverTrigger className={editableRowClass}>{content}</PopoverTrigger>
        <PopoverContent align="start" className="w-44 p-1">
          <div className="flex flex-col gap-0.5">
            {types.map((type) => (
              <button
                key={type.id}
                type="button"
                className={menuItemClass}
                onClick={() => onSetType(ticket.id, type.id)}
              >
                {type.name}
              </button>
            ))}
            {types.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No types</p>}
          </div>
        </PopoverContent>
      </Popover>
    </>
  );
};
