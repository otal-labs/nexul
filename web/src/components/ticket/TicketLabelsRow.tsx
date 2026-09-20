import { PlusIcon, TagIcon, XIcon } from "lucide-react";
import { useState } from "react";

import { Input } from "@/components/ui/input";
import { NoFillBadge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { labelDotColor } from "@/components/board/ticketTypeColor";
import { menuItemClass, rowClass } from "@/components/ticket/ticketPropertyRowStyle";
import { useFetchAllLabels } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/models/Ticket";

interface AddLabelPopoverProps {
  existingLabels: string[];
  onAdd: (label: string) => void;
}

const AddLabelPopover = ({ existingLabels, onAdd }: AddLabelPopoverProps) => {
  const { data: allLabels } = useFetchAllLabels();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const trimmed = search.trim();
  const candidates = (allLabels ?? []).filter((l) => !existingLabels.includes(l));
  const filtered = candidates.filter((l) => l.toLowerCase().includes(trimmed.toLowerCase()));
  const alreadyExists = [...candidates, ...existingLabels].some((l) => l.toLowerCase() === trimmed.toLowerCase());
  const showCreateRow = trimmed !== "" && !alreadyExists;

  const pick = (label: string) => {
    onAdd(label);
    setOpen(false);
    setSearch("");
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setSearch("");
      }}
    >
      <PopoverTrigger asChild>
        <button
          type="button"
          className="inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
        >
          <PlusIcon className="size-3" aria-hidden /> Add label
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-52 p-1.5">
        <Input
          autoFocus
          aria-label="Search labels"
          placeholder="Search or create…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="mb-1.5 h-8 text-xs"
        />
        <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">
          {filtered.map((label) => (
            <button key={label} type="button" className={menuItemClass} onClick={() => pick(label)}>
              <span className={cn("size-1.5 shrink-0 rounded-full", labelDotColor(label))} aria-hidden />
              {label}
            </button>
          ))}
          {showCreateRow && (
            <button type="button" className={menuItemClass} onClick={() => pick(trimmed)}>
              Create "{trimmed}"
            </button>
          )}
          {filtered.length === 0 && !showCreateRow && (
            <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};

interface TicketLabelsRowProps {
  ticket: Ticket;
  onAddLabel?: (ticketId: string, label: string) => Promise<void> | void;
  onRemoveLabel?: (ticketId: string, label: string) => Promise<void> | void;
}

export const TicketLabelsRow = ({ ticket, onAddLabel, onRemoveLabel }: TicketLabelsRowProps) => {
  const labels = ticket.labels ?? [];
  const canEdit = onAddLabel != null;

  return (
    <div className={cn(rowClass, "items-start")}>
      <TagIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <span className="sr-only">Labels</span>
      <div className="flex flex-1 flex-wrap items-center gap-x-1.5 gap-y-1">
        {labels.length === 0 && !canEdit && <span className="text-xs text-muted-foreground">None</span>}
        {labels.map((label) => (
          <span key={label} className="group/chip inline-flex items-center gap-0.5">
            <NoFillBadge color={labelDotColor(label)}>{label}</NoFillBadge>
            {onRemoveLabel && (
              <button
                type="button"
                aria-label={`Remove label ${label}`}
                onClick={() => onRemoveLabel(ticket.id, label)}
                className="text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard hover:text-foreground focus-visible:opacity-100 group-hover/chip:opacity-100"
              >
                <XIcon className="size-3" aria-hidden />
              </button>
            )}
          </span>
        ))}
        {canEdit && (
          <AddLabelPopover existingLabels={labels} onAdd={(label) => onAddLabel(ticket.id, label)} />
        )}
      </div>
    </div>
  );
};
