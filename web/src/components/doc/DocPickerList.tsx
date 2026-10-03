import { FileText } from "lucide-react";
import { useMemo, useState } from "react";

import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Input } from "@/components/ui/input";
import { useFetchDocsByProject } from "@/hooks/DocHooks";

const MAX_RESULTS = 50;

interface DocPickerListProps {
  projectId: string;
  excludeId: string;
  onSelect: (docId: string) => void;
}

// Searchable list of a project's open docs for a popover body; callers own the Popover and trigger around it.
export const DocPickerList = ({ projectId, excludeId, onSelect }: DocPickerListProps) => {
  const { data: docs } = useFetchDocsByProject(projectId);
  const [search, setSearch] = useState("");

  const options = useMemo(() => {
    const query = search.trim().toLowerCase();
    return (docs ?? [])
      .filter((d) => d.can_open && !d.archived && d.id !== excludeId)
      .filter((d) => d.title.toLowerCase().includes(query))
      .slice(0, MAX_RESULTS);
  }, [docs, excludeId, search]);

  return (
    <>
      <Input
        autoFocus
        aria-label="Search docs"
        placeholder="Search docs…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="mb-1.5 h-8 text-xs"
      />
      <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">
        {options.map((d) => (
          <button key={d.id} type="button" className={menuItemClass} onClick={() => onSelect(d.id)}>
            <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="truncate">{d.title}</span>
          </button>
        ))}
        {options.length === 0 && <p className="px-2.5 py-1.5 text-xs text-muted-foreground">No matches</p>}
      </div>
    </>
  );
};
