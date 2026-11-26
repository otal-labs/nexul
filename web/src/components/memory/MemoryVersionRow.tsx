import { personLabel } from "@nexul/client-core/person";

import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import type { HistoryVersionSummary } from "@/models/History";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MemoryVersionRowProps {
  version: HistoryVersionSummary;
  onOpen: (version: number) => void;
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (via: string, author: string): string => (via === "mcp" ? `Agent via ${author}` : author);

// One hairline row: the version, who saved it, and when (the current one says so instead, the header already dating it);
// it opens the history on that version, and the title it had sits in the tooltip, since it rarely changes between versions.
export const MemoryVersionRow = ({ version, onOpen }: MemoryVersionRowProps) => {
  const authorId = version.editor_ids[0] ?? "";
  const author = usePerson(authorId);

  return (
    <li title={version.title} className="border-b border-border last:border-b-0">
      <button
        type="button"
        onClick={() => onOpen(version.version)}
        className="flex h-8 w-full items-center gap-2 text-left text-xs transition-colors duration-150 ease-standard focus-visible:bg-accent/40 [&:hover>span]:text-foreground"
      >
        <span className="w-6 shrink-0 font-mono text-muted-foreground tabular-nums">v{version.version}</span>
        <span className={cn("min-w-0 flex-1 truncate", !authorId && "text-muted-foreground")}>{authorId ? authorLabel(version.via, personLabel(author)) : "No author recorded"}</span>
        <span className="shrink-0 font-mono text-muted-foreground tabular-nums">{version.current ? "current" : formatRelativeTime(version.created_at)}</span>
      </button>
    </li>
  );
};
