import { useState } from "react";

import { Button } from "@/components/ui/button";
import { SourcesProtoStep } from "@/components/memory/prototype/SourcesProtoStep";
import { DRAFTS } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import type { AnswerValue, QuestionItem } from "@/models/Question";

interface SourcesProtoQuestionProps {
  item: QuestionItem;
  progress: string;
  prevId: string | null;
}

const filled = (v: AnswerValue | undefined) => (v?.text?.trim() ?? "") !== "" || (v?.selected?.length ?? 0) > 0;

export const FromLine = ({ from, quote }: { from: string; quote: string }) => (
  <p className="mt-1 text-xs text-muted-foreground">
    From <span className="font-mono text-[11px]">{from}</span>: “{quote}”
  </p>
);

// The open row's body: the draft preselected with where it came from; Next confirms it as the answer.
export const SourcesProtoQuestion = ({ item, progress, prevId }: SourcesProtoQuestionProps) => {
  const answer = useSourcesProtoStore((s) => s.answers[item.id]);
  const drafted = useSourcesProtoStore((s) => s.drafted.includes(item.id));
  const save = useSourcesProtoStore((s) => s.save);
  const open = useSourcesProtoStore((s) => s.open);
  const source = drafted ? DRAFTS[item.id] : undefined;
  const [draft, setDraft] = useState<AnswerValue | undefined>(answer ?? source?.value);

  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {item.header && <p className="mt-1 text-xs text-muted-foreground">{item.header}</p>}
      {source && <FromLine from={source.from} quote={source.quote} />}
      <SourcesProtoStep item={item} draft={draft} drafted={source?.value} onDraft={setDraft} />
      <div className="mt-4 flex items-center justify-end gap-2">
        {prevId !== null && (
          <Button variant="ghost" size="sm" onClick={() => open(prevId)}>
            Back
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={() => save(item.id, null)}>
          Skip
        </Button>
        <Button size="sm" disabled={!filled(draft)} onClick={() => save(item.id, draft ?? null)}>
          Next
        </Button>
      </div>
    </div>
  );
};
