import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { RunsAsLine } from "@/components/autoplay/RunsAsLine";
import { runOnLabel, SUBJECT_RUN_ON, type AutoPlayDraft } from "@/models/AutoPlay";
import type { AutoPlayRunOn } from "@nexul/client-core/autoPlayText";

// A doc has no developer or tester and a stack has none of the three, so what is left to choose depends on the play.
export const RunOnSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const [subject, run_on, moment] = useWatch<AutoPlayDraft, ["subject", "run_on", "moment"]>({ name: ["subject", "run_on", "moment"] });

  return (
    <ComposerSection title="Runs as">
      <InlineSelect
        label="Runs as"
        value={run_on}
        options={SUBJECT_RUN_ON[subject].map((value) => ({ value, label: runOnLabel(value, subject, moment) }))}
        onChange={(value) => setValue("run_on", value as AutoPlayRunOn, { shouldDirty: true })}
        disabled={formState.disabled}
      />
      {run_on === "person" && <RunsAsLine />}
    </ComposerSection>
  );
};
