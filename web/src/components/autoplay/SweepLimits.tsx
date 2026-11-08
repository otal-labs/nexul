import { useFormContext, useWatch } from "react-hook-form";

import { Input } from "@/components/ui/input";
import type { AutoPlayDraft } from "@/models/AutoPlay";
import { MAX_SWEEP_RUNS, validZone, zoneName } from "@/models/Schedule";

// How much one sweep may take, and when it stops starting runs; an empty stop time runs until the tickets are done.
export const SweepLimits = () => {
  const { register, formState } = useFormContext<AutoPlayDraft>();
  const zone = useWatch<AutoPlayDraft, "timezone">({ name: "timezone" });
  return (
    <>
      <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
        At most
        <Input
          aria-label="Tickets per sweep"
          type="number"
          min={1}
          max={MAX_SWEEP_RUNS}
          className="technical h-8 w-16"
          disabled={formState.disabled}
          {...register("max_runs", { valueAsNumber: true })}
        />
        tickets a sweep
      </div>
      <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
        Start nothing after
        <Input aria-label="Stop at" type="time" className="technical h-8 w-28" disabled={formState.disabled} {...register("stop_at")} />
        {validZone(zone) && `${zoneName(zone)} time`}
      </div>
    </>
  );
};
