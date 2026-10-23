import { useFormContext, useWatch } from "react-hook-form";

import { ChosenComputers } from "@/components/autoplay/ChosenComputers";
import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { useListComputers } from "@/hooks/PairingHooks";
import { draftForChoice, RUN_ON_CHOICE_LABELS, runOnChoice, SUBJECT_RUN_ON, type AutoPlayDraft, type RunOnChoice } from "@/models/AutoPlay";

// A doc has no developer or tester and a stack has none of the three, so what is left to choose depends on the play.
export const RunOnSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const [subject, run_on, computer_ids, computers_picked] = useWatch<AutoPlayDraft, ["subject", "run_on", "computer_ids", "computers_picked"]>({
    name: ["subject", "run_on", "computer_ids", "computers_picked"],
  });
  const { data: own } = useListComputers();
  const choice = runOnChoice({ run_on, computer_ids, computers_picked });
  const choose = (next: RunOnChoice) => {
    if (next === "picked" && !own) return;
    const draft = draftForChoice(next, (own ?? []).map((c) => c.id));
    setValue("run_on", draft.run_on, { shouldDirty: true });
    setValue("computer_ids", draft.computer_ids, { shouldDirty: true });
    setValue("computers_picked", draft.computers_picked, { shouldDirty: true });
  };

  return (
    <ComposerSection title="Run on">
      <InlineSelect
        label="Run on"
        value={choice}
        placeholder="Pick computers"
        options={SUBJECT_RUN_ON[subject].map((value) => ({ value, label: RUN_ON_CHOICE_LABELS[value] }))}
        onChange={(value) => choose(value as RunOnChoice)}
        disabled={formState.disabled}
      />
      {run_on === "computer" && computers_picked && <ChosenComputers />}
    </ComposerSection>
  );
};
