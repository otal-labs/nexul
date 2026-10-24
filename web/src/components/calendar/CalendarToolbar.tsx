import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { viewDays, type CalendarView } from "@nexul/client-core/calendar";

import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useCalendarStore } from "@/stores/calendarStore";
import { periodTitle } from "@/utils/CalendarUtility";

// Step through the period, jump back to today, and switch between the month and the week.
export const CalendarToolbar = () => {
  const { view, anchor, setView, step, today } = useCalendarStore(
    useShallow((s) => ({ view: s.view, anchor: s.anchor, setView: s.setView, step: s.step, today: s.today })),
  );
  const unit = view === "month" ? "month" : "week";
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" className="size-8" aria-label={`Previous ${unit}`} onClick={() => step(-1)}>
          <ChevronLeftIcon aria-hidden />
        </Button>
        <Button variant="outline" size="icon" className="size-8" aria-label={`Next ${unit}`} onClick={() => step(1)}>
          <ChevronRightIcon aria-hidden />
        </Button>
        <Button variant="outline" size="sm" onClick={today}>
          Today
        </Button>
        <h2 className="ml-2 text-base font-semibold" aria-live="polite">
          {periodTitle(view, anchor, viewDays(view, anchor))}
        </h2>
      </div>
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        value={view}
        // Radix reports "" when the chosen view is pressed again; the view stands.
        onValueChange={(value) => value && setView(value as CalendarView)}
        aria-label="View"
      >
        <ToggleGroupItem value="month">Month</ToggleGroupItem>
        <ToggleGroupItem value="week">Week</ToggleGroupItem>
      </ToggleGroup>
    </div>
  );
};
