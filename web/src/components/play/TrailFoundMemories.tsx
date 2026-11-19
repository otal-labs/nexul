import { Fragment } from "react";
import { Link } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { memoryPath } from "@/models/Project";
import type { FoundMemory } from "@/models/Trail";

interface TrailFoundMemoriesProps {
  projectId: string;
  memories: FoundMemory[];
}

// The memories a run found by their when-to-use line and read, each opening its memory; a deleted one keeps its name, unlinked.
export const TrailFoundMemories = ({ projectId, memories }: TrailFoundMemoriesProps) => {
  const wsPath = useWorkspacePath();
  return (
    <span>
      {memories.map((memory, i) => (
        <Fragment key={memory.id}>
          {i > 0 && ", "}
          {!memory.deleted && (
            <Link to={wsPath(memoryPath(projectId, memory.id))} className="inset-focus rounded-sm underline decoration-muted-foreground underline-offset-4 transition-colors duration-150 ease-standard hover:decoration-foreground">
              {memory.title}
            </Link>
          )}
          {memory.deleted && <span className="text-muted-foreground">{memory.title}</span>}
        </Fragment>
      ))}
    </span>
  );
};
