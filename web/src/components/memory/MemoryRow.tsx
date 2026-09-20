import { BrainIcon } from "lucide-react";
import { Link } from "react-router";

import { formatUpdatedAgo } from "@/components/doc/docTime";
import type { Memory } from "@/models/Memory";
import { memoryPath } from "@/models/Project";
import { cn } from "@/lib/utils";

interface MemoryRowProps {
  memory: Memory;
  projectToken: string;
  /** Position in the (filtered) list; drives the entrance stagger, mirroring DocRow. */
  index: number;
}

const STAGGER_LIMIT = 8;
const STAGGER_STEP_MS = 25;

// Mirrors DocRow's hairline-row pattern; no checkbox column since memories have no bulk-permissions action.
export const MemoryRow = ({ memory, projectToken, index }: MemoryRowProps) => {
  const stagger = index < STAGGER_LIMIT;
  return (
    <li
      className={cn(stagger && "animate-in fade-in-0 slide-in-from-bottom-1 fill-mode-both duration-150 ease-out")}
      style={stagger ? { animationDelay: `${index * STAGGER_STEP_MS}ms` } : undefined}
    >
      <Link
        to={memoryPath(projectToken, memory.id)}
        className="flex w-full flex-col items-start gap-1 rounded-md px-1 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <span className="flex w-full items-center gap-2">
          <BrainIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span className="min-w-0 flex-1 truncate font-medium text-foreground">{memory.title}</span>
          {memory.always_included && (
            <span className="inline-flex shrink-0 items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
              always included
            </span>
          )}
        </span>
        {memory.when_to_use !== "" && (
          <span className="truncate text-xs text-muted-foreground">{memory.when_to_use}</span>
        )}
        <span className="font-mono text-xs text-muted-foreground tabular-nums">
          updated {formatUpdatedAgo(memory.updated_at)}
        </span>
      </Link>
    </li>
  );
};
