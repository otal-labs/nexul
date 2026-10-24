import { useRouter } from "expo-router";
import CalendarClock from "lucide-react-native/icons/calendar-clock";
import CalendarX from "lucide-react-native/icons/calendar-x";
import { Pressable, View } from "react-native";

import { isCancelled, type Meeting, type Occurrence } from "@nexul/client-core/calendar";

import { StatusTile } from "@/components/StatusTile";
import { Text } from "@/components/ui/text";
import { useMeetingPlace } from "@/hooks/CalendarHooks";
import { timeRange } from "@/lib/meetingTime";
import { cn } from "@/lib/utils";

interface AgendaMeetingRowProps {
  meeting: Meeting;
  occurrence: Occurrence;
}

// A cancelled meeting stays on the agenda, struck through with a red dot, so nobody turns up to it.
export const AgendaMeetingRow = ({ meeting, occurrence }: AgendaMeetingRowProps) => {
  const router = useRouter();
  const place = useMeetingPlace(meeting);
  const cancelled = isCancelled(meeting);
  const detail = [cancelled && "Cancelled", place?.label].filter(Boolean).join(" · ");
  return (
    <Pressable
      role="button"
      onPress={() => router.push(`/more/calendar/${meeting.id}?at=${encodeURIComponent(occurrence.starts_at)}`)}
      className="min-h-16 flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent"
    >
      <StatusTile icon={cancelled ? CalendarX : CalendarClock} dot={cancelled ? "bg-destructive" : undefined} />
      <View className="min-w-0 flex-1 gap-0.5">
        <Text numberOfLines={2} className={cn("text-[15px] leading-5", cancelled && "text-muted-foreground line-through")}>
          {meeting.title}
        </Text>
        {detail !== "" && (
          <Text numberOfLines={1} className="text-[13px] text-muted-foreground">
            {detail}
          </Text>
        )}
      </View>
      <Text className="font-mono text-xs text-muted-foreground">{timeRange(occurrence.starts_at, occurrence.ends_at)}</Text>
    </Pressable>
  );
};
