import { useFormContext, useWatch } from "react-hook-form";

import { Input } from "@/components/ui/input";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { AutoPlayDraft } from "@/models/AutoPlay";

// Whether its runs may overlap: one at a time is this auto play's own, or with a resource name, together with every auto play naming it.
export const OneAtATimeField = () => {
  const { register, setValue, formState } = useFormContext<AutoPlayDraft>();
  const on = useWatch<AutoPlayDraft, "one_at_a_time">({ name: "one_at_a_time" });
  return (
    <div className="space-y-1.5">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label="Overlap"
        value={on ? "one" : "overlap"}
        disabled={!!formState.disabled}
        // Radix reports "" when the chosen side is pressed again; the choice stands.
        onValueChange={(next) => next && setValue("one_at_a_time", next === "one", { shouldDirty: true })}
      >
        <ToggleGroupItem value="overlap">Can overlap</ToggleGroupItem>
        <ToggleGroupItem value="one">One at a time</ToggleGroupItem>
      </ToggleGroup>
      {on && (
        <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
          shared with auto plays naming
          <Input
            aria-label="Resource"
            placeholder="windows-vm"
            maxLength={40}
            spellCheck={false}
            className="technical h-8 w-40"
            disabled={formState.disabled}
            {...register("resource")}
          />
        </div>
      )}
      {on && <p className="text-xs text-muted-foreground">Auto plays naming the same resource never run side by side. Empty keeps only this one's runs apart.</p>}
    </div>
  );
};
