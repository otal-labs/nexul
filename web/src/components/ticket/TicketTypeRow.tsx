import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ticketTypeColor } from "@/components/board/ticketTypeColor";
import { TicketTypeIcon } from "@/components/board/ticketTypeIcon";
import {
  editableRowClass,
  menuItemClass,
  rowClass,
  rowIconClass,
  rowLabelClass,
  rowValueClass,
} from "@/components/ticket/ticketPropertyRowStyle";
import { TicketTypeOption } from "@/components/ticket/TicketTypeOption";
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

  const content = (
    <>
      <span className={rowIconClass}>
        <TicketTypeIcon typeName={current?.name ?? ""} className={cn("size-3.5", ticketTypeColor(current?.name ?? ""))} aria-hidden />
      </span>
      <span className={rowLabelClass}>Type</span>
      <span className={cn(rowValueClass, !current && "text-muted-foreground")}>{current?.name ?? "No type"}</span>
    </>
  );

  if (!onSetType) {
    return (
      <div className={rowClass}>{content}</div>
    );
  }

  return (
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
                <TicketTypeOption name={type.name} selected={type.id === current?.id} iconClassName={ticketTypeColor(type.name)} />
              </button>
            ))}
            {types.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No types</p>}
          </div>
        </PopoverContent>
      </Popover>
  );
};
