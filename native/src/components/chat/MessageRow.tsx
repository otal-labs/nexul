import { View } from "react-native";

import { MessageBody } from "@/components/chat/MessageBody";
import { Text } from "@/components/ui/text";
import { RelativeTime } from "@/components/RelativeTime";
import { cn } from "@/lib/utils";
import type { Message } from "@/models/Chat";

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
    </View>
  );
};
