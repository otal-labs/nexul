import type { RowDraft } from "@/models/InterviewAnswer";

interface InterviewDraftFromProps {
  draft: RowDraft;
}

// Where a draft came from: its sources, then the run's own line saying where in them.
export const InterviewDraftFrom = ({ draft }: InterviewDraftFromProps) => (
  <p className="mt-1 text-xs text-muted-foreground">
    From <span className="font-mono text-[11px]">{draft.from === "" ? "a removed source" : draft.from}</span>
    {draft.where !== "" && `: ${draft.where}`}
  </p>
);
