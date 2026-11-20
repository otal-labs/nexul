import type { Repeat } from "@nexul/client-core/calendar";

import { Text } from "@/components/ui/text";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

const REPEATS: { value: Repeat; label: string }[] = [
  { value: "none", label: "Once" },
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
];

// Once, or how often it posts again; a monthly one skips a month without its day.
export const ScheduleRepeatSwitch = ({ value, onChange }: { value: Repeat; onChange: (repeat: Repeat) => void }) => (
  <ToggleGroup
    type="single"
    variant="outline"
    value={value}
    // Pressing the active item reports an empty value; the current repeat stays.
    onValueChange={(next) => next && onChange(next as Repeat)}
    aria-label="Repeats"
    className="w-full"
  >
    {REPEATS.map((r, i) => (
      <ToggleGroupItem key={r.value} value={r.value} isFirst={i === 0} isLast={i === REPEATS.length - 1} className="h-11 flex-1" aria-label={r.label}>
        <Text>{r.label}</Text>
      </ToggleGroupItem>
    ))}
  </ToggleGroup>
);
