import { FileTextIcon, LockIcon } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { cn } from "@/lib/utils";
import type { DocListItem } from "@/models/Doc";

interface DocRowProps {
  doc: DocListItem;
  /** Position in the (filtered) list; drives the entrance stagger. */
  index: number;
  selected: boolean;
  onToggleSelect: () => void;
  onSelect: () => void;
}

// Stagger budget: up to 8 rows animate in; longer/virtualized lists appear instantly.
const STAGGER_LIMIT = 8;
const STAGGER_STEP_MS = 25;

// Row: hairline divide, not a card; checkbox-select, hover is a background lift only.
export const DocRow = ({ doc, index, selected, onToggleSelect, onSelect }: DocRowProps) => {
  const stagger = index < STAGGER_LIMIT;
  return (
    <li
      className={cn(stagger && "animate-in fade-in-0 slide-in-from-bottom-1 fill-mode-both duration-150 ease-out")}
      style={stagger ? { animationDelay: `${index * STAGGER_STEP_MS}ms` } : undefined}
    >
      <div className="flex w-full items-start gap-3 rounded-md px-1 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
        <Checkbox
          checked={selected}
          onCheckedChange={onToggleSelect}
          aria-label={`Select ${doc.title}`}
          className="mt-0.5"
        />
        <button
          type="button"
          className="flex min-w-0 flex-1 flex-col items-start gap-1 text-left"
          onClick={onSelect}
          disabled={!doc.can_open}
        >
          <span className="flex w-full items-center gap-2">
            <FileTextIcon className="size-4 shrink-0 text-muted-foreground" />
            <span className="min-w-0 truncate font-medium text-foreground">{doc.title}</span>
            {!doc.can_open && <LockIcon className="size-3.5 shrink-0 text-muted-foreground" />}
            {doc.archived && (
              <span className="inline-flex shrink-0 items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                archived
              </span>
            )}
          </span>
          <span className="font-mono text-xs text-muted-foreground tabular-nums">
            v{doc.version} · updated {formatUpdatedAgo(doc.updated_at)}
          </span>
        </button>
      </div>
    </li>
  );
};
