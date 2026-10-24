import { CalendarMonth } from "@/components/calendar/CalendarMonth";
import { CalendarToolbar } from "@/components/calendar/CalendarToolbar";
import { CalendarWeek } from "@/components/calendar/CalendarWeek";
import { MeetingSheet } from "@/components/calendar/MeetingSheet";
import { NewMeetingButton } from "@/components/calendar/NewMeetingButton";
import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { PageHeader } from "@/components/PageHeader";
import { useCalendarEntries } from "@/hooks/useCalendarEntries";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

// The viewer's meetings and the tickets due, by month or week; the grid stands while its entries load, so stepping never shows a spinner.
export const CalendarPage = () => {
  const workspaceCrumb = useWorkspaceCrumb();
  const { view, days, byDay, error } = useCalendarEntries();
  return (
    <Container className="space-y-5 py-8">
      <PageHeader crumbs={[workspaceCrumb]} title="Calendar" actions={<NewMeetingButton />} />
      <CalendarToolbar />
      {error && <ErrorDisplay error={error} title="Couldn't load the calendar." />}
      {view === "month" && <CalendarMonth days={days} byDay={byDay} />}
      {view === "week" && <CalendarWeek days={days} byDay={byDay} />}
      <MeetingSheet />
    </Container>
  );
};
