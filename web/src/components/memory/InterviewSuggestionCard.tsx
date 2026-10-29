import { ActionButton } from "@/components/action/ActionButton";
import { InterviewDraftFrom } from "@/components/memory/InterviewDraftFrom";
import { useDismissInterviewDraft } from "@/hooks/InterviewSourceHooks";
import { useSaveInterviewAnswer } from "@/hooks/MemoryHooks";
import { suggestionDiff, type InterviewRow, type RowDraft } from "@/models/InterviewAnswer";
import { answerLine } from "@nexul/client-core/questionChecklist";
import { cn } from "@/lib/utils";

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
    <div className="animate-in fade-in-0 slide-in-from-top-1 pt-2 pr-2.5 pb-5 pl-10.5 duration-200 ease-out">
      <p className="font-mono text-xs text-muted-foreground">{progress}</p>
      <div className="mt-2 space-y-2.5 rounded-lg border border-border bg-surface-2 px-3 py-2.5">
        <div>
          <p className="text-xs text-muted-foreground">Now</p>
          <p className="mt-0.5 flex gap-2 text-sm text-muted-foreground">
            <span aria-hidden className="w-2.5 shrink-0" />
            <span className="min-w-0">{answerLine(row.answer)}</span>
          </p>
        </div>
        <div>
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span aria-hidden className="size-1.5 rounded-full bg-warning" />
            Suggested
          </p>
          <ul className="mt-0.5 space-y-0.5 text-sm">
            {suggestionDiff(row.answer, draft.value).map((line) => (
              <li key={`${line.sign}${line.text}`} className="flex gap-2">
                <span aria-hidden className={cn("w-2.5 shrink-0 font-mono", line.sign === "+" ? "text-success" : "text-destructive")}>
                  {line.sign === "+" ? "+" : "−"}
                </span>
                <span className="sr-only">{line.sign === "+" ? "Add" : "Remove"}: </span>
                <span className={cn("min-w-0", line.sign === "-" && "text-muted-foreground line-through")}>{line.text}</span>
              </li>
            ))}
          </ul>
        </div>
        <InterviewDraftFrom draft={draft} label="Suggested from" />
      </div>
      <div className="mt-4 flex items-center justify-end gap-2">
        <ActionButton
          permission="memories:write"
          projectId={projectId}
          verb="dismiss suggestions"
          variant="ghost"
          size="sm"
          loading={dismiss.isPending}
          disabled={busy}
          onClick={() => dismiss.mutate(draft.id, { onSuccess: () => onMove(null) })}
        >
          Dismiss
        </ActionButton>
        <ActionButton permission="memories:write" projectId={projectId} verb="accept suggestions" size="sm" loading={save.isPending} disabled={busy} onClick={accept}>
          Accept
        </ActionButton>
      </div>
    </div>
  );
};
