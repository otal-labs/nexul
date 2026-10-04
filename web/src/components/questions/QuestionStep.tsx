import type { ReactNode } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";

import { DraftedTag, QuestionOptionRow } from "@/components/questions/QuestionOptionRow";
import { optionValue, type AnswerValue, type QuestionItem, type QuestionOption } from "@/models/Question";
import { cn } from "@/lib/utils";

interface QuestionStepProps {
  item: QuestionItem;
  // Unique per rendered step, so each option's label points at its own control.
  idPrefix: string;
  draft: AnswerValue | undefined;
  onDraft: (value: AnswerValue) => void;
  // What a drafting run proposed, tagged on its options and its text.
  drafted?: AnswerValue | undefined;
  // A line under the hint, such as where a draft came from.
  note?: ReactNode;
  // A question without options takes a growing text box, where Enter is a new line, instead of a one-line field.
  multiline?: boolean;
}

// toggle picks one value for a single choice and flips it in the set for a multi-select; either clears typed text.
export const toggleOption = (item: QuestionItem, draft: AnswerValue | undefined, value: string): AnswerValue => {
  if (!item.multi_select) return { selected: [value] };
  const current = draft?.selected ?? [];
  return { selected: current.includes(value) ? current.filter((v) => v !== value) : [...current, value] };
};

const listClass = "mt-2 divide-y divide-border overflow-hidden rounded-md border border-border";

// The answering part of one question, under whatever title its host gives it: the hint muted, the option rows, the free text.
export const QuestionStep = ({ item, idPrefix, draft, onDraft, drafted, note, multiline = false }: QuestionStepProps) => {
  const textBox = multiline && item.options.length === 0;
  const selected = draft?.selected ?? [];
  const isDrafted = (option: QuestionOption) => drafted?.selected?.includes(optionValue(option)) ?? false;
  const draftedText = (drafted?.text ?? "") !== "";
  const rowId = (i: number) => `${idPrefix}-option-${i}`;
  return (
    <div>
      {item.header && <p className="text-xs text-muted-foreground">{item.header}</p>}
      {note}
      {item.options.length > 0 && !item.multi_select && (
        <RadioGroup asChild className="gap-0" value={selected[0] ?? ""} onValueChange={(value) => onDraft(toggleOption(item, draft, value))}>
          <ul className={listClass}>
            {item.options.map((option, i) => (
              <QuestionOptionRow
                key={optionValue(option)}
                number={i + 1}
                option={option}
                checked={selected.includes(optionValue(option))}
                drafted={isDrafted(option)}
                htmlFor={rowId(i)}
                control={<RadioGroupItem id={rowId(i)} value={optionValue(option)} aria-label={option.label} />}
              />
            ))}
          </ul>
        </RadioGroup>
      )}
      {item.options.length > 0 && item.multi_select && (
        <ul className={listClass}>
          {item.options.map((option, i) => (
            <QuestionOptionRow
              key={optionValue(option)}
              number={i + 1}
              option={option}
              checked={selected.includes(optionValue(option))}
              drafted={isDrafted(option)}
              htmlFor={rowId(i)}
              control={
                <Checkbox
                  id={rowId(i)}
                  checked={selected.includes(optionValue(option))}
                  onCheckedChange={() => onDraft(toggleOption(item, draft, optionValue(option)))}
                  aria-label={option.label}
                />
              }
            />
          ))}
        </ul>
      )}
      {draftedText && (
        <div className="mt-2 flex justify-end">
          <DraftedTag />
        </div>
      )}
      {textBox && (
        <Textarea
          aria-label="Your answer"
          placeholder="Your answer"
          rows={1}
          value={draft?.text ?? ""}
          onChange={(e) => onDraft({ text: e.target.value })}
          className={cn("field-sizing-content min-h-8 resize-none py-1.5 text-sm", draftedText ? "mt-1" : "mt-2")}
        />
      )}
      {!textBox && (
        <Input
          type="text"
          aria-label={item.options.length > 0 ? "Something else" : "Your answer"}
          placeholder={item.options.length > 0 ? "Something else…" : "Your answer"}
          value={draft?.text ?? ""}
          onChange={(e) => onDraft({ text: e.target.value })}
          className={cn("h-8 text-sm", draftedText ? "mt-1" : "mt-2")}
        />
      )}
    </div>
  );
};
