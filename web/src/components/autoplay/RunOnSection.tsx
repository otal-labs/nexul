import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { RunsAsLine } from "@/components/autoplay/RunsAsLine";
import { runOnLabel, SUBJECT_RUN_ON, type AutoPlayDraft } from "@/models/AutoPlay";
import type { AutoPlayRunOn } from "@nexul/client-core/autoPlayText";

const ME = "__me";

// A doc has no developer or tester and a stack has none of the three, so what is left to choose depends on the play.
export const RunOnSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const [subject, run_on, moment, run_as, run_as_me] = useWatch<AutoPlayDraft, ["subject", "run_on", "moment", "run_as", "run_as_me"]>({
    name: ["subject", "run_on", "moment", "run_as", "run_as_me"],
  });
  const nobody = run_on === "person" && run_as === "" && !run_as_me;

  return (
    <ComposerSection title="Runs as">
      <div className="flex flex-wrap items-center gap-2">
      <InlineSelect
        label="Runs as"
        value={nobody ? "" : run_on}
        placeholder="Pick who it runs as"
        options={[
          ...(nobody ? [{ value: ME, label: "Run as me" }] : []),
          ...SUBJECT_RUN_ON[subject].map((value) => ({ value, label: runOnLabel(value, subject, moment) })),
        ]}
        onChange={(value) => {
          if (value === ME) {
            setValue("run_on", "person", { shouldDirty: true });
            setValue("run_as_me", true, { shouldDirty: true });
            return;
          }
          setValue("run_on", value as AutoPlayRunOn, { shouldDirty: true });
        }}
        disabled={formState.disabled}
      />
      {run_on === "person" && <RunsAsLine />}
      </div>
    </ComposerSection>
  );
};
