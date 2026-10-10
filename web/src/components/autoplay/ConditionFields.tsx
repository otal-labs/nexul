import { useFormContext, useWatch } from "react-hook-form";

import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { ValuesPicker } from "@/components/autoplay/ValuesPicker";
import { useAutoPlayValues } from "@/hooks/AutoPlayValueHooks";
import {
  blankRule,
  FIELD_LABELS,
  FIELD_OPS,
  opLabel,
  SUBJECT_FIELDS,
  takesValues,
  type AutoPlayDraft,
  type AutoPlayField,
  type AutoPlayOp,
} from "@/models/AutoPlay";

export type RulePath = `groups.${number}.rules.${number}` | `priority.rules.${number}.when.rules.${number}`;

// Field, operator, then values; a yes/no field and is set / is not set take no values.
export const ConditionFields = ({ path }: { path: RulePath }) => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const rule = useWatch<AutoPlayDraft, RulePath>({ name: path });
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const values = useAutoPlayValues(rule.field);
  const set = (next: typeof rule) => setValue(path, next, { shouldDirty: true });
  const name = FIELD_LABELS[rule.field];

  return (
    <div className="flex min-w-0 flex-1 items-center gap-1.5">
      <InlineSelect
        label="Field"
        value={rule.field}
        options={SUBJECT_FIELDS[subject].map((field) => ({ value: field, label: FIELD_LABELS[field] }))}
        onChange={(field) => set(blankRule(field as AutoPlayField))}
        disabled={formState.disabled}
        className="w-32 shrink-0 justify-between"
      />
      <InlineSelect
        label={`${name} operator`}
        value={rule.op}
        options={FIELD_OPS[rule.field].map((op) => ({ value: op, label: opLabel(rule.field, op) }))}
        onChange={(op) => set({ ...rule, op: op as AutoPlayOp, values: takesValues(op as AutoPlayOp) ? (rule.values ?? []) : [] })}
        disabled={formState.disabled}
        className="w-26 shrink-0 justify-between"
      />
      {takesValues(rule.op) && (
        <ValuesPicker
          label={`${name} values`}
          values={rule.values ?? []}
          field={values}
          onChange={(next) => set({ ...rule, values: next })}
          disabled={formState.disabled}
        />
      )}
    </div>
  );
};
