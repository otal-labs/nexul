import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection } from "@/components/autoplay/ComposerControls";
import { ValuesPicker } from "@/components/autoplay/ValuesPicker";
import { useAutoPlayMemoryValues } from "@/hooks/AutoPlayValueHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";

export const MemoriesSection = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const memoryIds = useWatch<AutoPlayDraft, "memory_ids">({ name: "memory_ids" });
  const field = useAutoPlayMemoryValues();

  return (
    <ComposerSection title="Memories">
      <div className="flex max-w-xl">
        <ValuesPicker
          label="Memories"
          values={memoryIds}
          field={field}
          onChange={(ids) => setValue("memory_ids", ids, { shouldDirty: true })}
          disabled={formState.disabled}
        />
      </div>
      <p className="text-xs text-muted-foreground">Read from each run's own project, on top of the memories every run gets.</p>
    </ComposerSection>
  );
};
