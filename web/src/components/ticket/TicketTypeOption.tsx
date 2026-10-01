import { CheckIcon } from "lucide-react";

import { TicketTypeIcon } from "@/components/board/ticketTypeIcon";
import { cn } from "@/lib/utils";

interface TicketTypeOptionProps {
  name: string;
  selected: boolean;
  iconClassName?: string;
}

export const TicketTypeOption = ({ name, selected, iconClassName }: TicketTypeOptionProps) => (
  <>
    <TicketTypeIcon typeName={name} className={cn("size-3.5 shrink-0", iconClassName)} aria-hidden />
    <span className="min-w-0 flex-1 truncate">{name}</span>
    {selected && <CheckIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
  </>
);
