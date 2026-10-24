import { useLocalSearchParams } from "expo-router";
import { ScrollView, View } from "react-native";

import { isCancelled } from "@nexul/client-core/calendar";

import { MeetingFacts } from "@/components/calendar/MeetingFacts";
import { MeetingRepliesSection } from "@/components/calendar/MeetingRepliesSection";
import { MeetingReplyBar } from "@/components/calendar/MeetingReplyBar";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { ScreenHeader } from "@/components/ScreenHeader";
import { Text } from "@/components/ui/text";
import { useFetchMeeting } from "@/hooks/CalendarHooks";
import { occurrenceEnd, timeRange } from "@/lib/meetingTime";
import { formatDayLabel } from "@/lib/time";

// One meeting, read-only but for the invitee's reply; `at` is the occurrence the agenda opened, else the row's own start.
export const MeetingScreen = () => {
  const { id, at } = useLocalSearchParams<{ id: string; at?: string }>();
  const { data: meeting, error, isPending } = useFetchMeeting(id);
  const startsAt = at ?? meeting?.starts_at ?? "";

  return (
    <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
      {isPending && <LoadingDisplay message="Loading the meeting" />}
      {error && <ErrorDisplay error={error} notFound="This meeting doesn't exist, or you're not in it." />}
      {meeting && (
        <ScreenHeader
          title={meeting.title}
          meta={`${formatDayLabel(startsAt)} · ${timeRange(startsAt, occurrenceEnd(meeting, startsAt))}`}
          className="pt-2"
        />
      )}
      {meeting && (
        <View className="gap-5 px-4">
          <MeetingFacts meeting={meeting} />
          {!isCancelled(meeting) && <MeetingReplyBar meeting={meeting} />}
          {meeting.notes !== "" && (
            <View className="gap-2">
              <Microheader>Notes</Microheader>
              <Text selectable className="text-[15px] leading-6">
                {meeting.notes}
              </Text>
            </View>
          )}
          <MeetingRepliesSection meeting={meeting} />
        </View>
      )}
    </ScrollView>
  );
};
