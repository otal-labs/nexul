import { useFormContext, useWatch } from "react-hook-form";

import { InlineSelect, QuietIcon, RemoveIcon } from "@/components/autoplay/ComposerControls";
import { ConditionRow } from "@/components/autoplay/ConditionRow";
import { blankRule, MATCH_OPTIONS, SUBJECT_FIELDS, type AutoPlayDraft, type AutoPlayMatch } from "@/models/AutoPlay";

interface NestedGroupEditorProps {
  index: number;
  onRemove: () => void;
}

// One indent in with its own sentence and no box; its last condition removed takes the group with it.
export const NestedGroupEditor = ({ index, onRemove }: NestedGroupEditorProps) => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const group = useWatch<AutoPlayDraft, `groups.${number}`>({ name: `groups.${index}` });
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const rules = group.rules;
  const writeRules = (next: typeof rules) => setValue(`groups.${index}.rules`, next, { shouldDirty: true });
  const removeRule = (at: number) => {
    if (rules.length === 1) {
      onRemove();
      return;
    }
    writeRules(rules.filter((_, i) => i !== at));
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <InlineSelect
          label="Group match"
          value={group.match}
          options={MATCH_OPTIONS}
          onChange={(match) => setValue(`groups.${index}.match`, match as AutoPlayMatch, { shouldDirty: true })}
          disabled={formState.disabled}
        />
        <p className="min-w-0 flex-1 text-sm text-muted-foreground">of these conditions must match:</p>
        {!formState.disabled && (
          <QuietIcon label="Add a condition to the group" onClick={() => writeRules([...rules, blankRule(SUBJECT_FIELDS[subject][0] ?? "project")])} />
        )}
        {!formState.disabled && <RemoveIcon label="Remove group" onClick={onRemove} />}
      </div>
      <div className="space-y-2 pl-5">
        {rules.map((_, i) => (
          <ConditionRow key={i} path={`groups.${index}.rules.${i}`} onRemove={() => removeRule(i)} />
        ))}
      </div>
    </div>
  );
};
