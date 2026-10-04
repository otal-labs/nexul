import { useState } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { InterviewSourceAddForm } from "@/components/memory/InterviewSourceAddForm";
import { InterviewSourceRow } from "@/components/memory/InterviewSourceRow";
import { useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { lastDraftedAt, sourceChange, type InterviewSource } from "@/models/InterviewSource";

interface InterviewSourceListProps {
  projectId: string;
  sources: InterviewSource[];
  readOnly: boolean;
}

const emptyClass = "border-0 px-1.5 py-1 text-left";

// The sources as hairline rows, then the inline add form or the button that opens it.
export const InterviewSourceList = ({ projectId, sources, readOnly }: InterviewSourceListProps) => {
  const [adding, setAdding] = useState(false);
  const drafting = useInterviewTrails(projectId).drafting;
  const since = drafting && lastDraftedAt(drafting);
  return (
    <div className="space-y-3">
      {sources.length === 0 && readOnly && <EmptyRow className={emptyClass}>No sources.</EmptyRow>}
      {sources.length === 0 && !readOnly && !adding && (
        <EmptyRow className={emptyClass}>
          Point the interview at what this project already has: a practices folder, a standards doc, the project it replaces.
        </EmptyRow>
      )}
      {sources.length > 0 && (
        <ul>
          {sources.map((source) => (
            <InterviewSourceRow key={source.id} source={source} change={sourceChange(source, since)} readOnly={readOnly} />
          ))}
        </ul>
      )}
      {adding && <InterviewSourceAddForm projectId={projectId} onClose={() => setAdding(false)} />}
      {!readOnly && !adding && (
        <Button variant="ghost" size="sm" onClick={() => setAdding(true)} className="text-muted-foreground">
          <Plus className="size-3.5" />
          Add source
        </Button>
      )}
    </div>
  );
};
