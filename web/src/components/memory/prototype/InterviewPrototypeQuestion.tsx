import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { QuestionOptionRow } from "@/components/play/QuestionOptionRow";
import { toggleOption } from "@/components/play/QuestionStep";
import type { ProtoQuestion } from "@/components/memory/prototype/InterviewPrototypeData";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { optionValue, type AnswerValue } from "@/models/Question";

interface InterviewPrototypeQuestionProps {
  question: ProtoQuestion;
  progress: string;
  first: boolean;
}

const listClass = "mt-3 divide-y divide-border overflow-hidden rounded-md border border-border";

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The open row's body, a copy of the question card's step laid flush under the row title, with Skip beside Back and Next.
export const InterviewPrototypeQuestion = ({ question, progress, first }: InterviewPrototypeQuestionProps) => {
  const saved = useInterviewPrototypeStore((s) => s.answers[question.id] ?? s.drafts[question.id]);
  const save = useInterviewPrototypeStore((s) => s.save);
  const back = useInterviewPrototypeStore((s) => s.back);
  const [draft, setDraft] = useState<AnswerValue | undefined>(saved);
  const selected = draft?.selected ?? [];
  const rowId = (i: number) => `proto-${question.id}-option-${i}`;
  const freeOnly = question.options.length === 0;

  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 pb-5 pl-8 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {question.why && (
        <p className="mt-1 text-sm text-muted-foreground">
          <span className="text-foreground">Why I'm asking: </span>
          {question.why}
        </p>
      )}
      {question.header && <p className="mt-1 text-sm text-muted-foreground">{question.header}</p>}
      {!freeOnly && !question.multi_select && (
        <RadioGroup asChild className="gap-0" value={selected[0] ?? ""} onValueChange={(value) => setDraft(toggleOption(question, draft, value))}>
          <ul className={listClass}>
            {question.options.map((option, i) => (
              <QuestionOptionRow
                key={optionValue(option)}
                number={i + 1}
                option={option}
                checked={selected.includes(optionValue(option))}
                htmlFor={rowId(i)}
                control={<RadioGroupItem id={rowId(i)} value={optionValue(option)} aria-label={option.label} />}
              />
            ))}
          </ul>
        </RadioGroup>
      )}
      {!freeOnly && question.multi_select && (
        <ul className={listClass}>
          {question.options.map((option, i) => (
            <QuestionOptionRow
              key={optionValue(option)}
              number={i + 1}
              option={option}
              checked={selected.includes(optionValue(option))}
              htmlFor={rowId(i)}
              control={
                <Checkbox
                  id={rowId(i)}
                  checked={selected.includes(optionValue(option))}
                  onCheckedChange={() => setDraft(toggleOption(question, draft, optionValue(option)))}
                  aria-label={option.label}
                />
              }
            />
          ))}
        </ul>
      )}
      {freeOnly && (
        <Textarea
          aria-label="Your answer"
          placeholder="Your answer"
          value={draft?.text ?? ""}
          onChange={(e) => setDraft({ text: e.target.value })}
          className="mt-3 min-h-20"
        />
      )}
      {!freeOnly && (
        <Input
          type="text"
          aria-label="Something else"
          placeholder="Something else…"
          value={draft?.text ?? ""}
          onChange={(e) => setDraft({ text: e.target.value })}
          className="mt-2 h-8 text-sm"
        />
      )}
      <div className="mt-4 flex items-center justify-end gap-2">
        {!first && (
          <Button variant="ghost" size="sm" onClick={() => back(question.id)}>
            Back
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={() => save(question.id, null)}>
          Skip
        </Button>
        <Button size="sm" disabled={!hasAnswer(draft)} onClick={() => draft && save(question.id, draft)}>
          Next
        </Button>
      </div>
    </div>
  );
};
