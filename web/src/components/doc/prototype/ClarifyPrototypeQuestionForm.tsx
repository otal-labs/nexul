import { useId, useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { QuestionStep } from "@/components/play/QuestionStep";
import { useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import type { AnswerValue, QuestionItem } from "@/models/Question";

interface ClarifyPrototypeQuestionFormProps {
  item: QuestionItem;
  why: string;
  progress: string;
  first: boolean;
}

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The interview's open-row body with a local save: options flush under the title, then Back, Skip, and Next.
export const ClarifyPrototypeQuestionForm = ({ item, why, progress, first }: ClarifyPrototypeQuestionFormProps) => {
  const saved = useClarifyPrototypeStore((s) => s.answers[item.id]);
  const save = useClarifyPrototypeStore((s) => s.save);
  const back = useClarifyPrototypeStore((s) => s.back);
  const [draft, setDraft] = useState<AnswerValue | undefined>(saved);
  const idPrefix = useId();
  const canNext = hasAnswer(draft);

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Enter" || !(e.target instanceof HTMLInputElement) || !canNext) return;
    e.preventDefault();
    save(item.id, draft ?? null);
  };

  return (
    <div onKeyDown={onKeyDown} className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {why !== "" && (
        <p className="mt-1 text-sm text-muted-foreground">
          <span className="text-foreground">Why we're asking: </span>
          {why}
        </p>
      )}
      <QuestionStep item={item} idPrefix={idPrefix} draft={draft} onDraft={setDraft} />
      <div className="mt-4 flex items-center justify-end gap-2">
        {!first && (
          <Button variant="ghost" size="sm" onClick={() => back(item.id)}>
            Back
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={() => save(item.id, null)}>
          Skip
        </Button>
        <Button size="sm" disabled={!canNext} onClick={() => save(item.id, draft ?? null)}>
          Next
        </Button>
      </div>
    </div>
  );
};
