import { ListPlusIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect, QuietIcon } from "@/components/autoplay/ComposerControls";
import { ConditionRow } from "@/components/autoplay/ConditionRow";
import { NestedGroupEditor } from "@/components/autoplay/NestedGroupEditor";
import { blankRule, MATCH_OPTIONS, SUBJECT_FIELDS, type AutoPlayDraft, type AutoPlayMatch, type DraftGroup } from "@/models/AutoPlay";

// "[All] of these N conditions must match:", a row per condition, nested groups one indent in.
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
  const addGroup = () => write([...groups, { match: "any", nested: true, rules: [blankRule(first)] }]);
  const count = groups.length;

  return (
    <ComposerSection title="If">
      <div className="space-y-2">
        <div className="flex items-center gap-1.5">
          {count > 0 && (
            <InlineSelect
              label="Match"
              value={match}
              options={MATCH_OPTIONS}
              onChange={(value) => setValue("match", value as AutoPlayMatch, { shouldDirty: true })}
              disabled={formState.disabled}
            />
          )}
          {count > 0 && (
            <p className="min-w-0 flex-1 text-sm text-muted-foreground">
              of {count === 1 ? "this" : "these"} <span className="font-medium text-foreground">{count}</span>{" "}
              {count === 1 ? "condition" : "conditions"} must match:
            </p>
          )}
          {count === 0 && <p className="min-w-0 flex-1 text-sm text-muted-foreground">No conditions: every {subject} at this moment runs it.</p>}
          {!formState.disabled && <QuietIcon label="Add a group" icon={ListPlusIcon} onClick={addGroup} />}
          {!formState.disabled && <QuietIcon label="Add a condition" onClick={addRule} />}
        </div>
        {count > 0 && (
          <div className="space-y-2 pl-5">
            {groups.map((group, i) => (
              <ConditionItem key={i} index={i} nested={group.nested} onRemove={() => without(i)} />
            ))}
          </div>
        )}
      </div>
    </ComposerSection>
  );
};

const ConditionItem = ({ index, nested, onRemove }: { index: number; nested: boolean; onRemove: () => void }) => (
  <div>
    {nested && <NestedGroupEditor index={index} onRemove={onRemove} />}
    {!nested && <ConditionRow path={`groups.${index}.rules.0`} onRemove={onRemove} />}
  </div>
);
