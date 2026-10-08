import { MemoryVersionRevertButton } from "@/components/memory/MemoryVersionRevertButton";
import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import type { MemoryVersion } from "@/models/MemoryVersion";
import { personLabel } from "@/models/Person";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MemoryVersionRowProps {
  memoryId: string;
  version: MemoryVersion;
  isCurrent: boolean;
  canRevert: boolean;
  /** "stacked" is the compact rail row: version and title on one line, who and when under it, revert revealed on hover. */
  layout?: "inline" | "stacked";
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (version: MemoryVersion, author: string): string =>
  version.author_via === "mcp" ? `Agent via ${author}` : author;

export const MemoryVersionRow = ({ memoryId, version, isCurrent, canRevert, layout = "inline" }: MemoryVersionRowProps) => {
  const stacked = layout === "stacked";
  const author = usePerson(version.author_id);
  const canRevertHere = !isCurrent && canRevert;

  return (
    <li
      className={cn(
        "flex gap-3",
        stacked
          ? "group items-start gap-2 rounded-md px-2 py-1.5 transition-colors duration-150 ease-standard hover:bg-accent/40"
          : "flex-wrap items-center px-4 py-3",
      )}
    >
      {!stacked && <span className="font-mono text-sm">v{version.version}</span>}
      <div className="min-w-0 flex-1">
        <p className={cn("truncate", stacked ? "text-xs" : "text-sm")}>
          {stacked && <span className="font-mono text-muted-foreground">v{version.version} </span>}
          {version.title}
        </p>
        <p className={cn("truncate text-xs text-muted-foreground", stacked && "font-mono text-[11px]")}>
          {authorLabel(version, version.author_id ? personLabel(author) : "Unknown")} · {formatRelativeTime(version.created_at)}
        </p>
      </div>
      {stacked && isCurrent && <span className="shrink-0 font-mono text-[11px] text-muted-foreground">current</span>}
      {canRevertHere && <MemoryVersionRevertButton memoryId={memoryId} version={version} compact={stacked} />}
    </li>
  );
};
