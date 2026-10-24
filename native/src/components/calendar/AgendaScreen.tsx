import CalendarDays from "lucide-react-native/icons/calendar-days";
import { useMemo } from "react";
import { View } from "react-native";

import { AgendaFeed } from "@/components/calendar/AgendaFeed";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ScreenHeader } from "@/components/ScreenHeader";
import { useFetchDueTickets, useFetchMeetings } from "@/hooks/CalendarHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { agendaDays } from "@/lib/agenda";

// Meetings and due tickets for the next 30 days, by the day they fall on in the phone's own zone.
export const AgendaScreen = () => {
  const workspace = useSelectedWorkspace();
  const meetings = useFetchMeetings();
  const due = useFetchDueTickets();
  const days = useMemo(
    () => (meetings.data && due.data ? agendaDays(meetings.data, due.data.items, new Date()) : undefined),
    [meetings.data, due.data],
  );
  const error = meetings.error ?? due.error;
  const header = <ScreenHeader eyebrow={workspace?.name} title="Calendar" meta="Next 30 days" className="pt-2" />;

  return (
    <View className="flex-1 bg-background">
      {!(days && days.length > 0) && header}
      {!days && !error && <LoadingDisplay message="Loading the calendar" />}
      {error && <ErrorDisplay error={error} />}
      {days && days.length === 0 && (
        <EmptyState icon={CalendarDays} title="Nothing coming up" message="Meetings you're in and tickets due in the next 30 days show here." />
      )}
      {days && days.length > 0 && (
        <AgendaFeed
          days={days}
          header={header}
          refreshing={meetings.isRefetching || due.isRefetching}
          onRefresh={() => void Promise.all([meetings.refetch(), due.refetch()])}
        />
      )}
    </View>
  );
};
