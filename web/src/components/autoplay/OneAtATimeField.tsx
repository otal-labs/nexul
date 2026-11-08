import { useFormContext, useWatch } from "react-hook-form";

import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import type { AutoPlayDraft } from "@/models/AutoPlay";

// One run at a time: this auto play's own, or with a resource name, together with every auto play naming it.
export const OneAtATimeField = () => {
  const { register, setValue, formState } = useFormContext<AutoPlayDraft>();
  const on = useWatch<AutoPlayDraft, "one_at_a_time">({ name: "one_at_a_time" });
  return (
    <div className="space-y-1.5">
      <label className="flex items-center gap-2 text-sm text-muted-foreground">
        <Switch
          checked={on}
          disabled={formState.disabled}
          onCheckedChange={(checked) => setValue("one_at_a_time", checked, { shouldDirty: true })}
        />
        One run at a time
      </label>
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
