import { PlusIcon, TagIcon, XIcon } from "lucide-react";
import { useState } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { Input } from "@/components/ui/input";
import { LabelChip } from "@/components/LabelChip";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { labelDotColor } from "@/components/board/ticketTypeColor";
import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { rowClass, rowIconClass, rowLabelClass } from "@/components/ticket/ticketPropertyRowStyle";
import { useFetchAllLabels, useFetchLabelColors } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import type { Ticket } from "@/models/Ticket";

interface AddLabelPopoverProps {
  projectId: string;
  existingLabels: string[];
  onAdd: (label: string) => void;
}

const AddLabelPopover = ({ projectId, existingLabels, onAdd }: AddLabelPopoverProps) => {
  const { data: allLabels } = useFetchAllLabels(projectId);
  const { data: labelColors } = useFetchLabelColors(projectId);
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
        <ActionButton
          permission="tickets:write"
          projectId={projectId}
          verb="label tickets"
          variant="ghost"
          size="sm"
          icon={PlusIcon}
          className="h-6 gap-1 px-1.5 text-xs text-muted-foreground"
        >
          Add label
        </ActionButton>
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
              <span className={cn("size-1.5 shrink-0 rounded-full", labelDotColor(label, labelColors?.[label]))} aria-hidden />
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
  onAddLabel: (ticketId: string, label: string) => Promise<void> | void;
  onRemoveLabel: (ticketId: string, label: string) => Promise<void> | void;
}

export const TicketLabelsRow = ({ ticket, onAddLabel, onRemoveLabel }: TicketLabelsRowProps) => {
  const labels = ticket.labels ?? [];
  const { data: labelColors } = useFetchLabelColors(ticket.project_id);

  return (
    <div className={cn(rowClass, "items-start")}>
      <span className={cn(rowIconClass, "mt-0.5")}>
        <TagIcon className="size-3.5" aria-hidden />
      </span>
      <span className={cn(rowLabelClass, "mt-0.5")}>Labels</span>
      <div className="flex flex-1 flex-wrap items-center gap-x-1.5 gap-y-1">
        {labels.length === 0 && <span className="text-xs text-muted-foreground">None</span>}
        {labels.map((label) => (
          <span key={label} className="group/chip inline-flex items-center gap-0.5">
            <LabelChip label={label} color={labelColors?.[label]} />
            <ActionButton
              permission="tickets:write"
              projectId={ticket.project_id}
              verb="label tickets"
              variant="ghost"
              size="icon"
              icon={XIcon}
              aria-label={`Remove label ${label}`}
              onClick={() => onRemoveLabel(ticket.id, label)}
              className="size-4 rounded-sm text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard focus-visible:opacity-100 group-hover/chip:opacity-100 [&_svg]:size-3"
            />
          </span>
        ))}
        <AddLabelPopover projectId={ticket.project_id} existingLabels={labels} onAdd={(label) => onAddLabel(ticket.id, label)} />
      </div>
    </div>
  );
};
