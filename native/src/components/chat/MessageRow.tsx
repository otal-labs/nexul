import { View } from "react-native";

import { MessageBody } from "@/components/chat/MessageBody";
import { Text } from "@/components/ui/text";
import { formatRelativeTime } from "@/lib/time";
import { cn } from "@/lib/utils";
import type { Message } from "@/models/Chat";

interface MessageRowProps {
  message: Message;
  authorLogin: string;
}

export const MessageRow = ({ message, authorLogin }: MessageRowProps) => {
  const isSystem = message.author_kind === "system";
  return (
    <View className={cn("gap-1 px-4 py-2", message.pending && "opacity-60")}>
      {isSystem && <Text className="text-xs italic text-muted-foreground">{message.body}</Text>}
      {!isSystem && (
        <View className="flex-row items-baseline gap-2">
          <Text numberOfLines={1} className="shrink text-sm font-semibold">
            {message.author_kind === "agent" ? "Agent" : authorLogin}
          </Text>
          <Text className="font-mono text-xs text-muted-foreground">{formatRelativeTime(message.created_at)}</Text>
          {message.edited_at && <Text className="text-xs text-muted-foreground">(edited)</Text>}
        </View>
      )}
      {!isSystem && <MessageBody body={message.body} />}
    </View>
  );
};
