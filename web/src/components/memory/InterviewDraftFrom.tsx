import { InterviewSourceChip } from "@/components/memory/InterviewSourceChip";
import type { RowDraft } from "@/models/InterviewAnswer";

interface InterviewDraftFromProps {
  draft: RowDraft;
  label: string;
}

// Where a draft came from: a chip per source that opens it, then the run's own sentence saying where in them.
export const InterviewDraftFrom = ({ draft, label }: InterviewDraftFromProps) => (
  <div className="mt-1.5 space-y-1.5">
    <p className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
      <span>{label}</span>
      {draft.cited.map((source) => (
        <InterviewSourceChip key={source.id} source={source} />
      ))}
      {draft.cited.length === 0 && <span>a removed source</span>}
    </p>
    {draft.where !== "" && <p className="text-xs text-muted-foreground">{draft.where}</p>}
  </div>
);
