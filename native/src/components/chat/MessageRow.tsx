import { View } from "react-native";

import { HandoffPill } from "@/components/chat/HandoffPill";
import { MessageBody } from "@/components/chat/MessageBody";
import { MessageReactions } from "@/components/chat/MessageReactions";
import { NoteFilePill } from "@/components/chat/NoteFilePill";
import { Text } from "@/components/ui/text";
import { RelativeTime } from "@/components/RelativeTime";
import { cn } from "@/lib/utils";
import { isNote, type Message } from "@/models/Chat";

interface MessageRowProps {
  message: Message;
  authorName: string;
  // The previous message is the same person's, so this one drops its name and time.
  continuation?: boolean;
}

export const MessageRow = ({ message, authorName, continuation = false }: MessageRowProps) => {
  const isSystem = message.author_kind === "system";
  return (
    <View className={cn("gap-1 px-4 pb-0.5", continuation ? "pt-0.5" : "pt-4", message.pending && "opacity-60")}>
      {isSystem && <Text className="text-xs italic text-muted-foreground">{message.body}</Text>}
      {!isSystem && !continuation && (
        <View className="flex-row items-baseline gap-2">
          <Text numberOfLines={1} className="shrink text-sm font-semibold">
            {message.author_kind === "agent" ? "Agent" : authorName}
          </Text>
          <Text className="font-mono text-xs text-muted-foreground"><RelativeTime iso={message.created_at} /></Text>
          {message.edited_at && <Text className="text-xs text-muted-foreground">(edited)</Text>}
        </View>
      )}
      {!isSystem && <MessageBody body={message.body} />}
      {isNote(message) && <NoteFilePill conversationId={message.conversation_id} attachmentId={message.attachment_id ?? ""} />}
      {message.handoffs && message.handoffs.length > 0 && (
        <View className="flex-row flex-wrap gap-x-1.5">
          {message.handoffs.map((handoff) => (
            <HandoffPill key={handoff.id} message={message} handoff={handoff} />
          ))}
        </View>
      )}
      {message.reactions && message.reactions.length > 0 && <MessageReactions reactions={message.reactions} />}
    </View>
  );
};
