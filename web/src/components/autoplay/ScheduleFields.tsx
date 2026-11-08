import { useFormContext, useWatch } from "react-hook-form";

import { Input } from "@/components/ui/input";
import type { AutoPlayDraft } from "@/models/AutoPlay";
import { describeSchedule, looksLikeCron, validZone } from "@/models/Schedule";

const ZONES = Intl.supportedValuesOf("timeZone");

// The cron and the zone it reads in, with the schedule in plain words under them as they are typed.
export const ScheduleFields = () => {
  const { register, formState } = useFormContext<AutoPlayDraft>();
  const cron = useWatch<AutoPlayDraft, "cron">({ name: "cron" });
  const zone = useWatch<AutoPlayDraft, "timezone">({ name: "timezone" });
  const fields = looksLikeCron(cron);
  const zoneOk = validZone(zone);

  return (
    <div className="space-y-1.5">
      <div className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
        <Input
          aria-label="Schedule"
          title="minute hour day month weekday"
          placeholder="0 23 * * *"
          aria-invalid={!fields || undefined}
          spellCheck={false}
          className="technical h-8 w-36"
          disabled={formState.disabled}
          {...register("cron")}
        />
        in
        <Input
          aria-label="Timezone"
          aria-invalid={!zoneOk || undefined}
          list="auto-play-zones"
          spellCheck={false}
          className="h-8 w-52"
          disabled={formState.disabled}
          {...register("timezone")}
        />
        <datalist id="auto-play-zones">
          {ZONES.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
      </div>
      {fields && zoneOk && <p className="text-xs text-muted-foreground">{describeSchedule(cron, zone)}</p>}
      {!fields && <p className="text-xs text-destructive">Five fields: minute, hour, day, month, weekday. Such as 0 23 * * *</p>}
      {fields && !zoneOk && <p className="text-xs text-destructive">Not a timezone. Such as Europe/London</p>}
    </div>
  );
};
