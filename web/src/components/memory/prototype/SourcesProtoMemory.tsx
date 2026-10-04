import type { ReactNode } from "react";
import { NotebookPen, Sparkles } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { InterviewMemoryView } from "@/components/memory/InterviewMemoryView";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { MEMORY_BODY } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import type { Memory } from "@/models/Memory";

const WRITTEN = new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString();

const MEMORY: Memory = {
  id: "proto-memory", workspace_id: "", project_id: "", kind: "interview", title: "Interview", when_to_use: "", body: MEMORY_BODY,
  always_included: true, footer: false, version: 3, created_by: "", created_at: WRITTEN, updated_by: "", updated_at: WRITTEN,
};

// The memory run's header line as InterviewRunLine draws it, mocked.
export const SourcesProtoMemoryLine = () => {
  const memory = useSourcesProtoStore((s) => s.memory);
  const changed = useSourcesProtoStore((s) => s.changed);
  const stale = memory && changed > 0;
  const detail = stale ? `${changed} ${changed === 1 ? "answer" : "answers"} changed` : "Written 2 days ago";
  return (
    <p className="flex min-w-0 items-center gap-2 text-sm">
      {stale && (
        <span role="img" aria-label="out of date" className="flex size-3.5 shrink-0 items-center justify-center">
          <span className="size-2 rounded-full bg-warning" />
        </span>
      )}
      {memory && !stale && <TrailStateIcon state="done" />}
      <span className="shrink-0 font-medium">{memory ? "Memory ready" : "Not generated yet"}</span>
      {memory && <span className="truncate text-muted-foreground">· {detail}</span>}
    </p>
  );
};

interface SourcesProtoMemoryProps {
  line?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
}

// The memory column: header line, Regenerate once a memory exists, whatever the variant puts above the memory, then it.
export const SourcesProtoMemory = ({ line, actions, children }: SourcesProtoMemoryProps) => {
  const memory = useSourcesProtoStore((s) => s.memory);
  const stale = useSourcesProtoStore((s) => s.memory && s.changed > 0);
  return (
    <section aria-label="Interview memory" className="min-w-0 space-y-4">
      <header className="flex min-h-9 flex-wrap items-center gap-3 border-b border-border pb-3">
        <div className="min-w-0 flex-1 basis-56">{line ?? <SourcesProtoMemoryLine />}</div>
        {actions}
        {memory && (
          <Button size="sm" variant={stale ? "default" : "outline"}>
            <Sparkles className="size-3.5" aria-hidden />
            {stale ? "Regenerate the memory" : "Regenerate"}
          </Button>
        )}
      </header>
      {children}
      {!memory && (
        <EmptyState size="compact" icon={NotebookPen} title="No memory yet" message="The agent writes the memory here once its follow-ups are answered." />
      )}
      {memory && <InterviewMemoryView memory={MEMORY} projectToken="proto" />}
    </section>
  );
};
