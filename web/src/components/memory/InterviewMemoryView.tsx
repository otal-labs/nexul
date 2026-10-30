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
    <article className="animate-in fade-in-0 slide-in-from-bottom-1 duration-200 ease-out">
      {/* The tab names it, and its title is the page's ("Interview"), so the line under the tab leads. */}
      <div className="mb-6 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <p className="text-sm text-muted-foreground">Every agent turn in this project reads it first. It can't be switched off.</p>
        <Link
          to={wsPath(memoryPath(projectToken, memory.id))}
          className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          Open in Memories →
        </Link>
      </div>
      <DocBodyView body={memory.body} />
    </article>
  );
};
