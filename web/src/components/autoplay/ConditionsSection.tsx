import { useFormContext, useWatch } from "react-hook-form";

import type { AutoPlayMatch } from "@nexul/client-core/autoPlayText";

import { ComposerSection, inlineActionClass, InlineSelect } from "@/components/autoplay/ComposerControls";
import { ConditionRow } from "@/components/autoplay/ConditionRow";
import { NestedGroupEditor } from "@/components/autoplay/NestedGroupEditor";
import { blankRule, MATCH_OPTIONS, SUBJECT_FIELDS, type AutoPlayDraft, type DraftGroup } from "@/models/AutoPlay";

// One level of conditions, a row each; an auto play that already has nested groups keeps its editor for them.
export const ConditionsSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const groups = useWatch<AutoPlayDraft, "groups">({ name: "groups" });
  const match = useWatch<AutoPlayDraft, "match">({ name: "match" });
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const first = SUBJECT_FIELDS[subject][0] ?? "project";
  const write = (next: DraftGroup[]) => setValue("groups", next, { shouldDirty: true });
  const without = (at: number) => write(groups.filter((_, i) => i !== at));
  // A new condition joins the plain rows, above the nested groups.
  const addRule = () => {
    const at = groups.some((g) => g.nested) ? groups.findIndex((g) => g.nested) : groups.length;
    write([...groups.slice(0, at), { match: "all", nested: false, rules: [blankRule(first)] }, ...groups.slice(at)]);
  };

  return (
    <ComposerSection title="Only if">
      {groups.length === 0 && <p className="pt-1.5 text-sm text-muted-foreground">No conditions</p>}
      {groups.map((group, i) => (
        <div key={i}>
          {group.nested && <NestedGroupEditor index={i} onRemove={() => without(i)} />}
          {!group.nested && <ConditionRow path={`groups.${i}.rules.0`} onRemove={() => without(i)} />}
        </div>
      ))}
      {!formState.disabled && (
        <div className="flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground">
          <button type="button" onClick={addRule} className={inlineActionClass}>
            + Add a condition
          </button>
          {groups.length > 1 && (
            <>
              <span aria-hidden>·</span>
              <InlineSelect
                label="Match"
                value={match}
                options={MATCH_OPTIONS}
                onChange={(value) => setValue("match", value as AutoPlayMatch, { shouldDirty: true })}
                className="h-6 px-1.5 text-xs"
              />
              <span>must match</span>
            </>
          )}
          {groups.length === 1 && <span>· all must match</span>}
        </div>
      )}
    </ComposerSection>
  );
};
