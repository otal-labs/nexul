import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";

import { QuestionOptionRow } from "@/components/play/QuestionOptionRow";
import { optionValue, type AnswerValue, type QuestionItem } from "@/models/Question";

interface QuestionStepProps {
  item: QuestionItem;
  index: number;
  total: number;
  draft: AnswerValue | undefined;
  onDraft: (value: AnswerValue) => void;
}

// toggle picks one value for a single choice and flips it in the set for a multi-select; either clears typed text.
export const toggleOption = (item: QuestionItem, draft: AnswerValue | undefined, value: string): AnswerValue => {
  if (!item.multi_select) return { selected: [value] };
  const current = draft?.selected ?? [];
  return { selected: current.includes(value) ? current.filter((v) => v !== value) : [...current, value] };
};

const listClass = "mt-2 divide-y divide-border overflow-hidden rounded-md border border-border";

// One question of the card: the progress line, the question as title, its hint muted, the option rows, and the free text.
export const QuestionStep = ({ item, index, total, draft, onDraft }: QuestionStepProps) => {
  const selected = draft?.selected ?? [];
  const rowId = (i: number) => `question-${index}-option-${i}`;
  return (
    <div>
      <p className="font-mono text-[11px] text-info">
        Question {index + 1} of {total}
      </p>
      <h4 className="mt-1 text-sm font-medium text-balance">{item.text}</h4>
      {item.header && <p className="text-xs text-muted-foreground">{item.header}</p>}
      {item.options.length > 0 && !item.multi_select && (
        <RadioGroup asChild value={selected[0] ?? ""} onValueChange={(value) => onDraft(toggleOption(item, draft, value))}>
          <ul className={listClass}>
            {item.options.map((option, i) => (
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
      {item.options.length > 0 && item.multi_select && (
        <ul className={listClass}>
          {item.options.map((option, i) => (
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
                  onCheckedChange={() => onDraft(toggleOption(item, draft, optionValue(option)))}
                  aria-label={option.label}
                />
              }
            />
          ))}
        </ul>
      )}
      <Input
        type="text"
        aria-label={item.options.length > 0 ? "Something else" : "Your answer"}
        placeholder={item.options.length > 0 ? "Something else…" : "Your answer"}
        value={draft?.text ?? ""}
        onChange={(e) => onDraft({ text: e.target.value })}
        className="mt-2 h-8 text-sm"
      />
    </div>
  );
};
