import { CheckIcon, Rows3 } from "lucide-react";
import { useState } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  editableRowClass,
  menuItemClass,
  rowIconClass,
  rowLabelClass,
  rowValueClass,
} from "@/components/ticket/ticketPropertyRowStyle";
import { useClearTicketCategory, useFetchProjectCategories, useMoveTicketToCategory } from "@/hooks/CategoryHooks";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/models/Ticket";

interface TicketCategoryRowProps {
  ticket: Ticket;
}

export const TicketCategoryRow = ({ ticket }: TicketCategoryRowProps) => {
  const [open, setOpen] = useState(false);
  const { data: categories = [] } = useFetchProjectCategories(ticket.project_id);
  const moveToCategory = useMoveTicketToCategory();
  const clearCategory = useClearTicketCategory();
  const current = categories.find((c) => c.id === ticket.category_id);
  const name = current?.name ?? "No category";

  const pick = (categoryId: string) => {
    setOpen(false);
    if (categoryId === ticket.category_id) return;
    if (categoryId === "") {
      clearCategory.mutate({ ticketId: ticket.id });
      return;
    }
    moveToCategory.mutate({ ticketId: ticket.id, categoryId });
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger className={editableRowClass} aria-label={`Category: ${name}`}>
        <span className={rowIconClass}>
          <Rows3 className="size-3.5" aria-hidden />
        </span>
        <span className={rowLabelClass}>Category</span>
        <span className={cn(rowValueClass, !current && "text-muted-foreground")}>{name}</span>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-48 p-1">
        <div className="flex flex-col gap-0.5">
          {categories.map((category) => (
            <button key={category.id} type="button" className={menuItemClass} onClick={() => pick(category.id)}>
              <span className="min-w-0 flex-1 truncate">{category.name}</span>
              {category.id === ticket.category_id && <CheckIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
            </button>
          ))}
          <button type="button" className={cn(menuItemClass, "text-muted-foreground")} onClick={() => pick("")}>
            No category
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
};
