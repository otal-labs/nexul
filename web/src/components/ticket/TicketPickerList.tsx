import { useMemo, useState } from "react";

import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Input } from "@/components/ui/input";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTickets } from "@/hooks/TicketHooks";

const MAX_RESULTS = 50;

interface TicketPickerListProps {
  excludeId: string;
  onSelect: (ticketId: string) => void;
}

// Searchable list of the workspace's tickets, every project, for a popover body; callers own the Popover and trigger.
export const TicketPickerList = ({ excludeId, onSelect }: TicketPickerListProps) => {
  const { data: tickets } = useFetchTickets();
  const { data: projects } = useFetchProjects();
  const [search, setSearch] = useState("");

  const options = useMemo(() => {
    const prefixes = new Map((projects ?? []).map((p) => [p.id, p.prefix]));
    const query = search.trim().toLowerCase();
    return (tickets ?? [])
      .filter((t) => t.id !== excludeId && prefixes.has(t.project_id))
      .map((t) => ({ id: t.id, key: `${prefixes.get(t.project_id)}-${t.number}`, title: t.title }))
      .filter((o) => `${o.key} ${o.title}`.toLowerCase().includes(query))
      .slice(0, MAX_RESULTS);
  }, [tickets, projects, excludeId, search]);

  return (
    <>
      <Input
        autoFocus
        aria-label="Search tickets"
        placeholder="Search tickets…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="mb-1.5 h-8 text-xs"
      />
      <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">
        {options.map((o) => (
          <button key={o.id} type="button" className={menuItemClass} onClick={() => onSelect(o.id)}>
            <span className="shrink-0 font-mono text-muted-foreground">{o.key}</span>
            <span className="truncate">{o.title}</span>
          </button>
        ))}
        {options.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>}
      </div>
    </>
  );
};
