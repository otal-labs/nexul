import { Link } from "react-router";

import { DocBodyView } from "@/components/doc/DocBodyView";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Memory } from "@/models/Memory";
import { memoryPath } from "@/models/Project";

interface InterviewMemoryViewProps {
  memory: Memory;
  projectToken: string;
}

// The interview memory as read; editing, versions, and attachments stay on its page under Memories.
export const InterviewMemoryView = ({ memory, projectToken }: InterviewMemoryViewProps) => {
  const wsPath = useWorkspacePath();
  return (
    <article className="animate-in fade-in-0 slide-in-from-bottom-1 rounded-lg bg-card px-6 py-8 shadow-card ring-1 ring-border duration-200 ease-out @3xl:px-12 @3xl:py-11">
      <div className="mb-2 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <h2 className="text-xl font-semibold tracking-tight">{memory.title}</h2>
        <Link
          to={wsPath(memoryPath(projectToken, memory.id))}
          className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          Open in Memories →
        </Link>
      </div>
      <p className="mb-6 text-sm text-muted-foreground">Always included in every agent turn in this project; it can't be switched off.</p>
      <DocBodyView body={memory.body} />
    </article>
  );
};
