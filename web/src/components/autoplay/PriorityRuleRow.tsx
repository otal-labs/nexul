import { useFormContext, useWatch } from "react-hook-form";

import { InlineSelect, RemoveIcon } from "@/components/autoplay/ComposerControls";
import { ConditionFields } from "@/components/autoplay/ConditionFields";
import { ConditionRow } from "@/components/autoplay/ConditionRow";
import { LEVEL_OPTIONS, type AutoPlayDraft, type AutoPlayLevel } from "@/models/AutoPlay";

interface PriorityRuleRowProps {
  index: number;
  onRemove: () => void;
}

// One level and the condition that sets it; a rule given more conditions elsewhere lists them under it, joined by and or or.
export const PriorityRuleRow = ({ index, onRemove }: PriorityRuleRowProps) => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const rule = useWatch<AutoPlayDraft, `priority.rules.${number}`>({ name: `priority.rules.${index}` });
  const extra = rule.when.rules.slice(1);
  const removeExtra = (at: number) =>
    setValue(`priority.rules.${index}.when.rules`, rule.when.rules.filter((_, i) => i !== at), { shouldDirty: true });

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <InlineSelect
          label="Level"
          value={rule.level}
          options={LEVEL_OPTIONS}
          onChange={(level) => setValue(`priority.rules.${index}.level`, level as AutoPlayLevel, { shouldDirty: true })}
          disabled={formState.disabled}
        />
        <span className="text-sm text-muted-foreground">if</span>
        <ConditionFields path={`priority.rules.${index}.when.rules.0`} />
        {!formState.disabled && <RemoveIcon label="Remove priority rule" onClick={onRemove} />}
      </div>
      {extra.length > 0 && (
        <div className="space-y-2 pl-5">
          {extra.map((_, i) => (
            <ConditionRow
              key={i}
              path={`priority.rules.${index}.when.rules.${i + 1}`}
              lead={rule.when.match === "all" ? "and" : "or"}
              onRemove={() => removeExtra(i + 1)}
            />
          ))}
        </div>
      )}
    </div>
  );
};
