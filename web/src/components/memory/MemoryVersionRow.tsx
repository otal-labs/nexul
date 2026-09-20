import { RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRevertMemory } from "@/hooks/MemoryHooks";
import type { MemoryVersion } from "@/models/MemoryVersion";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MemoryVersionRowProps {
  memoryId: string;
  version: MemoryVersion;
  isCurrent: boolean;
  canRevert: boolean;
}

// authorLabel names who saved the version; an MCP-tagged save reads "Agent via <user>" (ADR 0049).
const authorLabel = (version: MemoryVersion): string => {
  const author = version.author_id || "Unknown";
  return version.author_via === "mcp" ? `Agent via ${author}` : author;
};

export const MemoryVersionRow = ({ memoryId, version, isCurrent, canRevert }: MemoryVersionRowProps) => {
  const revert = useRevertMemory();
  const { open: confirm } = useConfirmationDialog();

  const onRevert = async () => {
    const ok = await confirm({
      title: `Revert to version ${version.version}?`,
      message: "This appends a new version with this version's content; nothing in the history is deleted.",
    });
    if (ok) revert.mutate({ id: memoryId, version: version.version });
  };

  return (
    <li className="flex flex-wrap items-center gap-3 px-4 py-3">
      <span className="font-mono text-sm">v{version.version}</span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">{version.title}</p>
        <p className="truncate text-xs text-muted-foreground">
          {authorLabel(version)} · {formatRelativeTime(version.created_at)}
        </p>
      </div>
      {!isCurrent && canRevert && (
        <Button type="button" variant="outline" size="sm" onClick={onRevert} disabled={revert.isPending}>
          <RotateCcw className="size-4" />
          Revert
        </Button>
      )}
    </li>
  );
};
