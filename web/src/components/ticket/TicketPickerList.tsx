import { useMemo, useState } from "react";

import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Input } from "@/components/ui/input";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useSearchWorkspace } from "@/hooks/SearchHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";

const MAX_RESULTS = 50;

interface TicketPickerListProps {
  projectId: string;
  excludeId: string;
  onSelect: (ticketId: string) => void;
}

// Searchable ticket list for a popover body: the open project's tickets, then the whole workspace's once two characters are typed.
export const TicketPickerList = ({ projectId, excludeId, onSelect }: TicketPickerListProps) => {
  const [search, setSearch] = useState("");
  const searching = search.trim().length >= 2;
  const { data: tickets } = useFetchTicketsByProject(searching ? undefined : projectId);
  const { data: projects } = useFetchProjects();
  const { data: found } = useSearchWorkspace(search, "ticket", MAX_RESULTS);

  const options = useMemo(() => {
    if (searching) {
      return (found?.hits ?? [])
        .filter((h) => h.id !== excludeId)
        .map((h) => ({ id: h.id, key: h.hint ?? "", title: h.title }))
        .slice(0, MAX_RESULTS);
    }
    const prefixes = new Map((projects ?? []).map((p) => [p.id, p.prefix]));
    const query = search.trim().toLowerCase();
    return (tickets ?? [])
      .filter((t) => t.id !== excludeId && prefixes.has(t.project_id))
      .map((t) => ({ id: t.id, key: `${prefixes.get(t.project_id)}-${t.number}`, title: t.title }))
      .filter((o) => `${o.key} ${o.title}`.toLowerCase().includes(query))
      .slice(0, MAX_RESULTS);
  }, [searching, found, tickets, projects, excludeId, search]);

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
        {(!searching || found) && options.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>}
      </div>
    </>
  );
};
