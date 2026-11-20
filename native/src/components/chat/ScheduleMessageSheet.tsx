import { useLocalSearchParams } from "expo-router";
import { View } from "react-native";

import { ScheduleMessageForm } from "@/components/chat/ScheduleMessageForm";
import { SheetTitle } from "@/components/SheetTitle";
import { Text } from "@/components/ui/text";
import { useFetchScheduledMessages } from "@/hooks/ScheduledMessageHooks";

type ScheduleParams = { conversationId: string; body?: string; id?: string };

// Schedules the composer's text, or with an id changes one of your scheduled messages; it posts as you at the time picked.
export const ScheduleMessageSheet = () => {
  const { conversationId, body = "", id } = useLocalSearchParams<ScheduleParams>();
  const { data: scheduled } = useFetchScheduledMessages(id ? conversationId : undefined);
  const editing = scheduled?.find((s) => s.id === id);
  const gone = !!id && !!scheduled && !editing;
  const ready = !id || !!editing;
  return (
    <View className="gap-4 bg-popover px-5 pb-6">
      <SheetTitle title={id ? "Edit scheduled message" : "Schedule message"} className="px-0 pb-0" />
      {gone && <Text className="text-body text-muted-foreground">This scheduled message already posted or was cancelled.</Text>}
      {ready && <ScheduleMessageForm conversationId={conversationId} body={body} editing={editing} />}
    </View>
  );
};
