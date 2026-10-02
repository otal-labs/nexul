import { RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRevertMemory } from "@/hooks/MemoryHooks";
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
  /** "stacked" puts the title under the version number and the revert button under the author, for a column too narrow for one line. */
  layout?: "inline" | "stacked";
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (version: MemoryVersion, author: string): string =>
  version.author_via === "mcp" ? `Agent via ${author}` : author;

export const MemoryVersionRow = ({ memoryId, version, isCurrent, canRevert, layout = "inline" }: MemoryVersionRowProps) => {
  const stacked = layout === "stacked";
  const revert = useRevertMemory();
  const author = usePerson(version.author_id);
  const { open: confirm } = useConfirmationDialog();

  const onRevert = async () => {
    const ok = await confirm({
      title: `Revert to version ${version.version}?`,
      message: "This appends a new version with this version's content; nothing in the history is deleted.",
    });
    if (ok) revert.mutate({ id: memoryId, version: version.version });
  };

  return (
    <li className={cn("flex gap-3", stacked ? "flex-col gap-1 px-3 py-2" : "flex-wrap items-center px-4 py-3")}>
      <span className="font-mono text-sm">v{version.version}</span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">{version.title}</p>
        <p className="truncate text-xs text-muted-foreground">
          {authorLabel(version, version.author_id ? personLabel(author) : "Unknown")} · {formatRelativeTime(version.created_at)}
        </p>
      </div>
      {!isCurrent && canRevert && (
        <Button type="button" variant="outline" size="sm" className={cn(stacked && "self-start")} onClick={onRevert} loading={revert.isPending}>
          <RotateCcw className="size-4" />
          Revert
        </Button>
      )}
    </li>
  );
};
