import { PlusIcon } from "lucide-react";
import { useState } from "react";

import { InterviewQuestionRow } from "@/components/settings/InterviewQuestionRow";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { joinTemplate, splitTemplate } from "@/utils/InterviewTemplateUtility";

interface InterviewQuestionListProps {
  body: string;
  onChange: (body: string) => void;
  readOnly: boolean;
}

// The template as the questions it asks; each row edits its own slice of the markdown, so nothing outside it changes.
export const InterviewQuestionList = ({ body, onChange, readOnly }: InterviewQuestionListProps) => {
  const [editing, setEditing] = useState<number | null>(null);
  const parts = splitTemplate(body);
  const numbers = parts.reduce<number[]>((acc, part) => [...acc, (acc.at(-1) ?? 0) + (part.trim() === "" ? 0 : 1)], []);

  const replace = (index: number, part: string) => onChange(joinTemplate(parts.map((p, i) => (i === index ? part : p))));
  const remove = (index: number) => {
    setEditing(null);
    onChange(joinTemplate(parts.filter((_, i) => i !== index)));
  };
  const add = () => {
    onChange(`${body.replace(/\n*$/, "")}\n\n## New question\n`);
    setEditing(parts.length);
  };

  return (
    <div className="space-y-3">
      <ol className={settingsListClass}>
        {parts.map(
          (part, index) =>
            part.trim() !== "" && (
              <InterviewQuestionRow
                key={index}
                number={numbers[index]!}
                part={part}
                readOnly={readOnly}
                editing={editing === index}
                onEditing={(open) => setEditing(open ? index : null)}
                onChange={(next) => replace(index, next)}
                onRemove={() => remove(index)}
              />
            ),
        )}
      </ol>
      {!readOnly && (
        <Button type="button" variant="ghost" size="sm" className="-ml-3" onClick={add}>
          <PlusIcon />
          Add question
        </Button>
      )}
    </div>
  );
};
