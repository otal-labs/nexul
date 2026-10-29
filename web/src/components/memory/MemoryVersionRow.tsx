import { personLabel } from "@nexul/client-core/person";

import { usePerson } from "@/hooks/PeopleHooks";
import type { HistoryVersionSummary } from "@/models/History";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MemoryVersionRowProps {
  version: HistoryVersionSummary;
  onOpen: (version: number) => void;
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (via: string, author: string): string => (via === "mcp" ? `Agent via ${author}` : author);

// The version and who saved it over when, opening the history on it; the title it had sits in the tooltip, since it rarely changes between versions.
export const MemoryVersionRow = ({ version, onOpen }: MemoryVersionRowProps) => {
  const authorId = version.editor_ids[0] ?? "";
  const author = usePerson(authorId);

  return (
    <li title={version.title}>
      <button
        type="button"
        onClick={() => onOpen(version.version)}
        className="flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:bg-accent/40"
      >
        <span className="min-w-0 flex-1 text-xs">
          <span className="block truncate">
            <span className="font-mono text-muted-foreground tabular-nums">v{version.version}</span>{" "}
            {authorLabel(version.via, authorId ? personLabel(author) : "Unknown")}
          </span>
          <span className="block truncate font-mono text-muted-foreground tabular-nums">{formatRelativeTime(version.created_at)}</span>
        </span>
        {version.current && <span className="shrink-0 font-mono text-xs text-muted-foreground">current</span>}
      </button>
    </li>
  );
};
