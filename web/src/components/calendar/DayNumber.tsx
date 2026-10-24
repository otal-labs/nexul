import { cn } from "@/lib/utils";
import { formatDayOfMonth } from "@/utils/CalendarUtility";

interface DayNumberProps {
  day: Date;
  today: boolean;
}

// Today is the day in an ink disc; ink, not the accent, which is for what you act on.
export const DayNumber = ({ day, today }: DayNumberProps) => (
  <span
    aria-current={today ? "date" : undefined}
    className={cn(
      "inline-flex size-6 items-center justify-center rounded-full font-mono text-xs tabular-nums",
      today && "bg-foreground font-semibold text-background",
    )}
  >
    {formatDayOfMonth(day)}
  </span>
);
