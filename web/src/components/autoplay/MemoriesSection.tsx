import { useFormContext, useWatch } from "react-hook-form";

import { ComposerSection } from "@/components/autoplay/ComposerControls";
import { ValuesPicker } from "@/components/autoplay/ValuesPicker";
import { useAutoPlayMemoryValues } from "@/hooks/AutoPlayValueHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";

const hint = (included: number, pickable: number): string => {
  if (included === 1 && pickable === 0) return "Your only memory is always included, so every run gets it. Switch it off in Memories to pick it here.";
  if (included > 0 && pickable === 0) return `All ${included} memories are always included, so every run gets them. Switch one off in Memories to pick it here.`;
  if (included > 0) return `Every run gets ${included} always-included ${included === 1 ? "memory" : "memories"}. Picks here add more, read from each run's own project.`;
  return "Read from each run's own project.";
};

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
      <p className="text-xs text-muted-foreground">{hint(field.included, field.options.length)}</p>
    </ComposerSection>
  );
};
