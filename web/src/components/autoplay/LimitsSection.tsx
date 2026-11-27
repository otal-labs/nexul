import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { OneAtATimeField } from "@/components/autoplay/OneAtATimeField";
import { SweepLimits } from "@/components/autoplay/SweepLimits";
import { LIMIT_PRESETS, repeatLabel, type AutoPlayDraft } from "@/models/AutoPlay";

// How often the same ticket or doc may start it, and whether its runs go one at a time.
export const LimitsSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const moment = useWatch<AutoPlayDraft, "moment">({ name: "moment" });
  const minutes = useWatch<AutoPlayDraft, "once_within_minutes">({ name: "once_within_minutes" });
  // A window set some other way (an agent's 2 hours) stays offered, so opening and saving never changes it.
  const choices = LIMIT_PRESETS.includes(minutes) ? LIMIT_PRESETS : [...LIMIT_PRESETS, minutes].sort((a, b) => a - b);

  return (
    <>
      {moment === "schedule" && (
        <ComposerSection title="Sweep">
          <SweepLimits />
        </ComposerSection>
      )}
      <ComposerSection title="Repeat">
        <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
          <InlineSelect
            label="Repeat"
            value={String(minutes)}
            options={choices.map((value) => ({ value: String(value), label: repeatLabel(value) }))}
            onChange={(value) => setValue("once_within_minutes", Number(value), { shouldDirty: true })}
            disabled={formState.disabled}
          />
          per {subject}
        </div>
      </ComposerSection>
      <ComposerSection title="One at a time">
        <OneAtATimeField />
      </ComposerSection>
    </>
  );
};
