import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { RUN_ON_LABELS, type AutoPlayDraft, type AutoPlayRunOn } from "@/models/AutoPlay";

const RUN_ON_OPTIONS = (Object.keys(RUN_ON_LABELS) as AutoPlayRunOn[]).map((value) => ({ value, label: RUN_ON_LABELS[value] }));

// A doc has no developer or tester, so a doc play's auto play always runs on whoever caused the moment.
export const RunOnSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const runOn = useWatch<AutoPlayDraft, "run_on">({ name: "run_on" });

  return (
    <ComposerSection title="Run on">
      {subject === "doc" && <p className="text-sm">{RUN_ON_LABELS.causer}</p>}
      {subject === "ticket" && (
        <InlineSelect
          label="Run on"
          value={runOn}
          options={RUN_ON_OPTIONS}
          onChange={(value) => setValue("run_on", value as AutoPlayRunOn, { shouldDirty: true })}
          disabled={formState.disabled}
        />
      )}
    </ComposerSection>
  );
};
