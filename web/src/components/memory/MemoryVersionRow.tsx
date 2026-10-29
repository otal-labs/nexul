import { personLabel } from "@nexul/client-core/person";

import { MemoryVersionRevertButton } from "@/components/memory/MemoryVersionRevertButton";
import { usePerson } from "@/hooks/PeopleHooks";
import type { MemoryVersion } from "@/models/MemoryVersion";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MemoryVersionRowProps {
  memoryId: string;
  version: MemoryVersion;
  isCurrent: boolean;
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (version: MemoryVersion, author: string): string =>
  version.author_via === "mcp" ? `Agent via ${author}` : author;

// The version and who saved it over when; the title it had sits in the tooltip, since it rarely changes between versions.
export const MemoryVersionRow = ({ memoryId, version, isCurrent }: MemoryVersionRowProps) => {
  const author = usePerson(version.author_id);

  return (
    <li
      title={version.title}
      className="group flex items-start gap-2 rounded-md px-2 py-1.5 transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <div className="min-w-0 flex-1 text-xs">
        <p className="truncate">
          <span className="font-mono text-muted-foreground tabular-nums">v{version.version}</span>{" "}
          {authorLabel(version, version.author_id ? personLabel(author) : "Unknown")}
        </p>
        <p className="truncate font-mono text-muted-foreground tabular-nums">{formatRelativeTime(version.created_at)}</p>
      </div>
      {isCurrent && <span className="shrink-0 font-mono text-xs text-muted-foreground">current</span>}
      {!isCurrent && <MemoryVersionRevertButton memoryId={memoryId} version={version} />}
    </li>
  );
};
