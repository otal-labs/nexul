import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { ChosenComputers } from "@/components/autoplay/ChosenComputers";
import { RUN_ON_LABELS, SUBJECT_RUN_ON, type AutoPlayDraft, type AutoPlayRunOn } from "@/models/AutoPlay";

// A doc has no developer or tester and a stack has none of the three, so what is left to choose depends on the play.
export const RunOnSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const runOn = useWatch<AutoPlayDraft, "run_on">({ name: "run_on" });
  const choices = SUBJECT_RUN_ON[subject];

  return (
    <ComposerSection title="Run on">
      {choices.length === 1 && <p className="text-sm">{RUN_ON_LABELS[runOn]}</p>}
      {choices.length > 1 && (
        <InlineSelect
          label="Run on"
          value={runOn}
          options={choices.map((value) => ({ value, label: RUN_ON_LABELS[value] }))}
          onChange={(value) => setValue("run_on", value as AutoPlayRunOn, { shouldDirty: true })}
          disabled={formState.disabled}
        />
      )}
      {runOn === "computer" && <ChosenComputers />}
    </ComposerSection>
  );
};
