import { useLocalSearchParams } from "expo-router";
import { ScrollView } from "react-native";

import { SheetBody } from "@/components/SheetBody";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ScheduledMessageRow } from "@/components/chat/ScheduledMessageRow";
import { SheetTitle } from "@/components/SheetTitle";
import { useFetchScheduledMessages } from "@/hooks/ScheduledMessageHooks";

// Your messages waiting to post in this conversation, soonest first; nobody else sees them until they do.
export const ScheduledMessagesSheet = () => {
  const { conversationId } = useLocalSearchParams<{ conversationId: string }>();
  const { data: scheduled, error, isPending } = useFetchScheduledMessages(conversationId);
  return (
    <SheetBody className="gap-4 px-5">
      <SheetTitle title="Scheduled messages" className="px-0 pb-0" />
      {isPending && <LoadingDisplay message="Loading scheduled messages" />}
      {error && <ErrorDisplay error={error} className="px-0" />}
      {scheduled && scheduled.length === 0 && <EmptyRow message="Nothing scheduled here." />}
      {scheduled && scheduled.length > 0 && (
        <ScrollView className="max-h-[480px]" contentContainerClassName="gap-3">
          {scheduled.map((s) => <ScheduledMessageRow key={s.id} scheduled={s} />)}
        </ScrollView>
      )}
    </SheetBody>
  );
};
