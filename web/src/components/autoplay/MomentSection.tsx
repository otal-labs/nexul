import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { MOMENT_LABELS, SUBJECT_MOMENTS, type AutoPlayDraft, type AutoPlayMoment } from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS, PLAY_STAGES, type PlayStage } from "@/models/Play";

const STAGE_OPTIONS = PLAY_STAGES.map((stage) => ({ value: stage, label: PLAY_STAGE_LABELS[stage] }));

export const MomentSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const subject = useWatch<AutoPlayDraft, "subject">({ name: "subject" });
  const moment = useWatch<AutoPlayDraft, "moment">({ name: "moment" });
  const stage = useWatch<AutoPlayDraft, "moment_stage">({ name: "moment_stage" });

  return (
    <ComposerSection title="When">
      <div className="flex flex-wrap items-center gap-1.5">
        <InlineSelect
          label="Moment"
          value={moment}
          options={SUBJECT_MOMENTS[subject].map((value) => ({ value, label: MOMENT_LABELS[value] }))}
          onChange={(value) => setValue("moment", value as AutoPlayMoment, { shouldDirty: true })}
          disabled={formState.disabled}
        />
        {moment === "ticket.entered_stage" && (
          <InlineSelect
            label="Stage"
            value={stage}
            options={STAGE_OPTIONS}
            onChange={(value) => setValue("moment_stage", value as PlayStage, { shouldDirty: true })}
            disabled={formState.disabled}
          />
        )}
      </div>
      {moment === "doc.changed" && (
        <p className="text-xs text-muted-foreground">Fires once edits stop for 10 minutes, never for an agent's edits.</p>
      )}
    </ComposerSection>
  );
};
