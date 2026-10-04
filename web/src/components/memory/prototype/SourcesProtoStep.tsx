import type { ReactNode } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { toggleOption } from "@/components/play/QuestionStep";
import { optionValue, type AnswerValue, type QuestionItem, type QuestionOption } from "@/models/Question";
import { cn } from "@/lib/utils";

const DraftedTag = () => <span className="shrink-0 font-mono text-[10px] text-muted-foreground uppercase">Drafted</span>;

interface OptionRowProps {
  number: number;
  option: QuestionOption;
  checked: boolean;
  drafted: boolean;
  control: ReactNode;
  htmlFor: string;
}

// QuestionOptionRow with room for the Drafted tag trailing the label.
const OptionRow = ({ number, option, checked, drafted, control, htmlFor }: OptionRowProps) => (
  <li className={cn("transition-colors duration-150 ease-standard hover:bg-accent/40", checked && "bg-accent/40")}>
    <label htmlFor={htmlFor} className="flex cursor-pointer items-start gap-2.5 px-3 py-2">
      <span className="mt-0.5 w-3 shrink-0 font-mono text-[10px] text-muted-foreground tabular-nums">{number}</span>
      <span className="mt-0.5 flex shrink-0 items-center">{control}</span>
      <span className="min-w-0 flex-1 text-sm">{option.label}</span>
      {drafted && <DraftedTag />}
    </label>
  </li>
);

interface SourcesProtoStepProps {
  item: QuestionItem;
  draft: AnswerValue | undefined;
  // What the drafting run picked, tagged on its options and its text.
  drafted: AnswerValue | undefined;
  onDraft: (value: AnswerValue) => void;
}

const listClass = "mt-2 divide-y divide-border overflow-hidden rounded-md border border-border";

// QuestionStep's options and free text, with the run's draft preselected and tagged.
export const SourcesProtoStep = ({ item, draft, drafted, onDraft }: SourcesProtoStepProps) => {
  const selected = draft?.selected ?? [];
  const id = (i: number) => `${item.id}-option-${i}`;
  const row = (option: QuestionOption, i: number, control: ReactNode) => (
    <OptionRow
      key={optionValue(option)}
      number={i + 1}
      option={option}
      checked={selected.includes(optionValue(option))}
      drafted={drafted?.selected?.includes(optionValue(option)) ?? false}
      htmlFor={id(i)}
      control={control}
    />
  );
  const draftedText = (drafted?.text ?? "") !== "";
  return (
    <div>
      {item.options.length > 0 && !item.multi_select && (
        <RadioGroup asChild className="gap-0" value={selected[0] ?? ""} onValueChange={(v) => onDraft(toggleOption(item, draft, v))}>
          <ul className={listClass}>
            {item.options.map((o, i) => row(o, i, <RadioGroupItem id={id(i)} value={optionValue(o)} aria-label={o.label} />))}
          </ul>
        </RadioGroup>
      )}
      {item.options.length > 0 && item.multi_select && (
        <ul className={listClass}>
          {item.options.map((o, i) =>
            row(
              o,
              i,
              <Checkbox id={id(i)} checked={selected.includes(optionValue(o))} onCheckedChange={() => onDraft(toggleOption(item, draft, optionValue(o)))} aria-label={o.label} />,
            ),
          )}
        </ul>
      )}
      {draftedText && (
        <div className="mt-2 flex justify-end">
          <DraftedTag />
        </div>
      )}
      {item.options.length > 0 && (
        <Input
          type="text"
          aria-label="Something else"
          placeholder="Something else…"
          value={draft?.text ?? ""}
          onChange={(e) => onDraft({ ...draft, text: e.target.value })}
          className={cn("h-8 text-sm", draftedText ? "mt-1" : "mt-2")}
        />
      )}
      {item.options.length === 0 && (
        <Textarea
          aria-label="Your answer"
          placeholder="Your answer"
          value={draft?.text ?? ""}
          onChange={(e) => onDraft({ text: e.target.value })}
          className={cn("field-sizing-content min-h-8 resize-none py-1.5 text-sm", draftedText ? "mt-1" : "mt-2")}
        />
      )}
    </div>
  );
};
