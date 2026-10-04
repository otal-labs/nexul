import { Button } from "@/components/ui/button";
import { InterviewDraftFrom } from "@/components/memory/InterviewDraftFrom";
import { useDismissInterviewDraft } from "@/hooks/InterviewSourceHooks";
import { useSaveInterviewAnswer } from "@/hooks/MemoryHooks";
import { answerLine, valueLine, type InterviewRow, type RowDraft } from "@/models/InterviewAnswer";

interface InterviewSuggestionCardProps {
  row: InterviewRow;
  draft: RowDraft;
  projectId: string;
  progress: string;
  onMove: (key: string | null) => void;
}

// An answered question a source now disagrees with: the confirmed answer, the suggestion under it, Dismiss or Accept.
export const InterviewSuggestionCard = ({ row, draft, projectId, progress, onMove }: InterviewSuggestionCardProps) => {
  const save = useSaveInterviewAnswer();
  const dismiss = useDismissInterviewDraft(projectId);
  const busy = save.isPending || dismiss.isPending;

  const accept = () =>
    save.mutate(
      { project_id: projectId, round: row.round, question: row.item.text, selected: draft.value.selected ?? [], text: draft.value.text ?? "", skip: false },
      { onSuccess: () => onMove(null) },
    );

  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      <div className="mt-2 divide-y divide-border rounded-md border border-border">
        <div className="px-3 py-2">
          <p className="text-xs text-muted-foreground">Your answer</p>
          <p className="mt-0.5 text-sm">{answerLine(row.answer)}</p>
        </div>
        <div className="px-3 py-2">
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span aria-hidden className="size-1.5 rounded-full bg-warning" />
            Suggested change
          </p>
          <p className="mt-0.5 text-sm">{valueLine(draft.value)}</p>
          <InterviewDraftFrom draft={draft} />
        </div>
      </div>
      <div className="mt-4 flex items-center justify-end gap-2">
        <Button variant="ghost" size="sm" loading={dismiss.isPending} disabled={busy} onClick={() => dismiss.mutate(draft.id, { onSuccess: () => onMove(null) })}>
          Dismiss
        </Button>
        <Button size="sm" loading={save.isPending} disabled={busy} onClick={accept}>
          Accept
        </Button>
      </div>
    </div>
  );
};
