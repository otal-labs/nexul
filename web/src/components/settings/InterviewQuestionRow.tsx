import { PencilIcon } from "lucide-react";

import { SettingsHint } from "@/components/settings/SettingsHint";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { readQuestion } from "@/utils/InterviewTemplateUtility";

interface InterviewQuestionRowProps {
  number: number;
  part: string;
  readOnly: boolean;
  editing: boolean;
  onEditing: (editing: boolean) => void;
  onChange: (part: string) => void;
  onRemove: () => void;
}

// One question as the interview asks it; Edit opens its own markdown in place, so the rest of the list stays scannable.
export const InterviewQuestionRow = ({ number, part, readOnly, editing, onEditing, onChange, onRemove }: InterviewQuestionRowProps) => {
  const question = readQuestion(part);
  const title = question?.title || part.trim();
  return (
    <li className="flex gap-4 py-4">
      <span className="w-5 shrink-0 pt-px font-mono text-xs text-muted-foreground tabular-nums">{String(number).padStart(2, "0")}</span>
      {!editing && (
        <div className="min-w-0 flex-1 space-y-1">
          <p className="text-sm font-medium text-pretty text-foreground">{title}</p>
          {question?.hint && <SettingsHint>{question.hint}</SettingsHint>}
          {question && question.options.length > 0 && (
            <p className="pt-1 text-xs text-pretty text-muted-foreground">
              {question.multiSelect ? "Any of: " : "One of: "}
              {question.options.join(" · ")}
            </p>
          )}
        </div>
      )}
      {editing && (
        <div className="min-w-0 flex-1 animate-in space-y-2 fade-in-0 slide-in-from-top-1 duration-200 ease-out">
          <Textarea
            aria-label={`Question ${number} as markdown`}
            value={part}
            onChange={(e) => onChange(e.target.value)}
            autoFocus
            spellCheck={false}
            className="field-sizing-content min-h-24 resize-none bg-surface-2 font-mono text-xs leading-relaxed"
          />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" size="sm" className="hover:text-destructive" onClick={onRemove}>
              Remove
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => onEditing(false)}>
              Done
            </Button>
          </div>
        </div>
      )}
      {!editing && !readOnly && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={`Edit question ${number}`}
          className="-my-1 size-8 shrink-0 text-muted-foreground"
          onClick={() => onEditing(true)}
        >
          <PencilIcon className="size-3.5" />
        </Button>
      )}
    </li>
  );
};
