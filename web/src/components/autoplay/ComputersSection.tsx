import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

import { ruleMode, ruleValues, withMode, withValues, type ComputerRuleMode } from "@nexul/client-core/autoPlayComputers";

import { ComputerPills } from "@/components/agent/ComputerPills";
import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { ValuesPicker } from "@/components/autoplay/ValuesPicker";
import { useAutoPlayPlaces } from "@/hooks/useAutoPlayPlaces";
import { useListComputers } from "@/hooks/PairingHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";

const MODE_LABELS: Record<ComputerRuleMode, string> = { any: "Any computer", labelled: "Labelled", not_labelled: "Not labelled", only: "Only these" };

// Which of the person's computers its runs may land on: by label, or by name when it runs as a fixed person.
export const ComputersSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const [run_on, run_as, run_as_me, match, groups, rule] = useWatch<AutoPlayDraft, ["run_on", "run_as", "run_as_me", "match", "groups", "computer_rule"]>({
    name: ["run_on", "run_as", "run_as_me", "match", "groups", "computer_rule"],
  });
  // An empty list reads as "any", so the picked kind is kept here until values arrive.
  const [mode, setMode] = useState(() => ruleMode(rule));
  const { data: computers } = useListComputers();
  const places = useAutoPlayPlaces({ run_on, run_as, run_as_me, conditions: { match, groups } }, rule);
  const nobody = places?.kind === "role" && places.role === "nobody";
  const ready = (computers ?? []).filter((c) => !c.unfinished);
  const labels = [...new Set(ready.flatMap((c) => [...(c.labels ?? []), ...(c.derived_labels ?? [])]))].sort();
  const options = mode === "only" ? ready.map((c) => ({ value: c.id, label: c.name })) : labels.map((label) => ({ value: label, label }));
  const write = (next: typeof rule) => setValue("computer_rule", next, { shouldDirty: true });

  return (
    <ComposerSection title="On computers">
      {nobody && <p className="pt-1.5 text-sm text-muted-foreground">Set Runs as first</p>}
      {!nobody && (
      <div className="flex min-w-0 flex-wrap items-center gap-1.5">
        <InlineSelect
          label="Computer choice"
          value={mode}
          options={(Object.keys(MODE_LABELS) as ComputerRuleMode[]).map((value) => ({ value, label: MODE_LABELS[value], disabled: value === "only" && run_on !== "person" }))}
          onChange={(value) => {
            setMode(value as ComputerRuleMode);
            write(withMode(value as ComputerRuleMode));
          }}
          disabled={formState.disabled}
        />
        {mode !== "any" && (
          <ValuesPicker
            label={mode === "only" ? "Computers" : "Labels"}
            values={ruleValues(rule, mode)}
            field={{ options, label: (value) => options.find((o) => o.value === value)?.label ?? value, empty: mode === "only" ? "No computers yet" : "No computer has a label yet" }}
            onChange={(values) => write(withValues(rule, mode, values))}
            disabled={formState.disabled}
          />
        )}
      </div>
      )}
      {places && !nobody && (
        <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
          <span>Right now that's</span>
          <ComputerPills places={places} showExcluded />
        </div>
      )}
    </ComposerSection>
  );
};
