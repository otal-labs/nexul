import { useRouter } from "expo-router";
import CalendarClock from "lucide-react-native/icons/calendar-clock";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { useFetchScheduledMessages } from "@/hooks/ScheduledMessageHooks";

// A line above the composer while you have messages waiting to post here; it opens them.
export const ScheduledMessagesLine = ({ conversationId }: { conversationId: string }) => {
  const router = useRouter();
  const { data: scheduled } = useFetchScheduledMessages(conversationId);
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const count = scheduled?.length ?? 0;
  const label = count === 1 ? "1 scheduled message" : `${count} scheduled messages`;
  return (
    count > 0 && (
      <Pressable
        role="button"
        aria-label={label}
        onPress={() => router.push({ pathname: "/chat/scheduled", params: { conversationId } })}
        className="min-h-9 flex-row items-center gap-1.5 self-start rounded-md px-4 active:bg-accent"
      >
        <CalendarClock color={String(muted)} size={14} />
        <Text className="text-row text-muted-foreground">{label}</Text>
      </Pressable>
    )
  );
};
