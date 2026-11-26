import { cn } from "@/lib/utils";
import { formatDayOfMonth } from "@/utils/CalendarUtility";

interface DayNumberProps {
  day: Date;
  today: boolean;
}

// Today is the day in an ink disc (ink, not the accent); other days start on the disc's left edge, so one digit or two line up.
export const DayNumber = ({ day, today }: DayNumberProps) => (
  <span
    aria-current={today ? "date" : undefined}
    className={cn(
      "inline-flex h-6 shrink-0 items-center font-mono text-row tabular-nums",
      today && "w-6 justify-center rounded-full bg-foreground font-semibold text-background",
    )}
  >
    {formatDayOfMonth(day)}
  </span>
);
