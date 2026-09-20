import { useMemo, useState } from "react";
import { SearchIcon } from "lucide-react";

import { DocRow } from "@/components/doc/DocRow";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Button } from "@/components/ui/button";
import type { DocListItem } from "@/models/Doc";

interface DocsFeedProps {
  docs: DocListItem[];
  selected: string[];
  onToggleSelect: (id: string) => void;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onPermissions: () => void;
}

export const DocsFeed = ({
  docs,
  selected,
  onToggleSelect,
  onSelect,
  onCreate,
  onPermissions,
}: DocsFeedProps) => {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q.length > 0 ? docs.filter((doc) => doc.title.toLowerCase().includes(q)) : docs;
  }, [docs, query]);

  return (
    <div>
      {docs.length > 0 && (
        <div className="relative mb-3 max-w-xs">
          <SearchIcon
            className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search docs…"
            aria-label="Search docs"
            className="h-9 w-full rounded-md border border-border bg-card pl-8 pr-3 text-sm text-foreground placeholder:text-muted-foreground transition-colors duration-150 ease-standard focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
          />
        </div>
      )}
      <div className="flex items-center justify-between border-b border-border py-2.5">
        <p className="font-mono text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase">
          {selected.length > 0
            ? `${selected.length} selected`
            : `${filtered.length} ${filtered.length === 1 ? "doc" : "docs"}`}
        </p>
        <div className="flex items-center gap-2">
          {selected.length > 0 && (
            <Button variant="outline" size="sm" onClick={onPermissions}>
              Permissions ({selected.length})
            </Button>
          )}
          <Button size="sm" onClick={onCreate}>
            New doc
          </Button>
        </div>
      </div>
      {docs.length === 0 && <NoDataDisplay message="No docs yet." />}
      {docs.length > 0 && filtered.length === 0 && (
        <NoDataDisplay message={`No docs match "${query.trim()}".`} />
      )}
      {filtered.length > 0 && (
        <ul className="divide-y divide-border">
          {filtered.map((doc, index) => (
            <DocRow
              key={doc.id}
              doc={doc}
              index={index}
              selected={selected.includes(doc.id)}
              onToggleSelect={() => onToggleSelect(doc.id)}
              onSelect={() => onSelect(doc.id)}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
