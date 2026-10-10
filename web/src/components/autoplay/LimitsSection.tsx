import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { LIMIT_PRESETS, limitLabel, type AutoPlayDraft } from "@/models/AutoPlay";

export const LimitsSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const minutes = useWatch<AutoPlayDraft, "once_within_minutes">({ name: "once_within_minutes" });
  // A window set some other way (an agent's 2 hours) stays offered, so opening and saving never changes it.
  const choices = LIMIT_PRESETS.includes(minutes) ? LIMIT_PRESETS : [...LIMIT_PRESETS, minutes].sort((a, b) => a - b);

  return (
    <ComposerSection title="Limits">
      <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
        At most once per {subject} every{" "}
        <InlineSelect
          label="Limit"
          value={String(minutes)}
          options={choices.map((value) => ({ value: String(value), label: limitLabel(value) }))}
          onChange={(value) => setValue("once_within_minutes", Number(value), { shouldDirty: true })}
          disabled={formState.disabled}
        />
      </div>
    </ComposerSection>
  );
};
