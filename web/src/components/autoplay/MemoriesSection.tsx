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
      <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
        Also read
        <ValuesPicker
          label="Memories"
          values={memoryIds}
          field={field}
          onChange={(ids) => setValue("memory_ids", ids, { shouldDirty: true })}
          disabled={formState.disabled}
        />
      </div>
      <p className="text-xs text-muted-foreground">
        Each run reads the picked memories from its own project, beside the ones always included.
      </p>
    </ComposerSection>
  );
};
