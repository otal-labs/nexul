import { useFormContext, useWatch } from "react-hook-form";

import type { AutoPlayLevel } from "@nexul/client-core/autoPlayText";

import { ComposerSection, inlineActionClass, InlineSelect } from "@/components/autoplay/ComposerControls";
import { PriorityRuleRow } from "@/components/autoplay/PriorityRuleRow";
import { blankRule, LEVEL_OPTIONS, SUBJECT_FIELDS, type AutoPlayDraft } from "@/models/AutoPlay";

// "[High] if <condition>" rows, checked in order, then "Otherwise [Normal]".
export const PrioritySection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const rules = useWatch<AutoPlayDraft, "priority.rules">({ name: "priority.rules" });
  const otherwise = useWatch<AutoPlayDraft, "priority.otherwise">({ name: "priority.otherwise" });
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const write = (next: typeof rules) => setValue("priority.rules", next, { shouldDirty: true });
  const add = () =>
    write([...rules, { level: "high", when: { match: "all", rules: [blankRule(SUBJECT_FIELDS[subject][0] ?? "project")] } }]);

  return (
    <ComposerSection title="Priority">
      <div className="space-y-2">
        {rules.map((_, i) => (
          <PriorityRuleRow key={i} index={i} onRemove={() => write(rules.filter((_, at) => at !== i))} />
        ))}
        <div className="flex items-center gap-1.5">
          {rules.length > 0 && <span className="text-sm text-muted-foreground">Otherwise</span>}
          <InlineSelect
            label="Otherwise level"
            value={otherwise}
            options={LEVEL_OPTIONS}
            onChange={(level) => setValue("priority.otherwise", level as AutoPlayLevel, { shouldDirty: true })}
            disabled={formState.disabled}
          />
          {!formState.disabled && (
            <button type="button" onClick={add} className={inlineActionClass}>
              + Add a priority rule
            </button>
          )}
        </div>
      </div>
    </ComposerSection>
  );
};
