import { View } from "react-native";

import { BotAvatar } from "@/components/chat/BotAvatar";
import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import type { Message } from "@/models/Chat";

// The name and avatar the post was sent with; the BOT tag keeps a post calling itself "GitHub" from passing for a person.
export const BotMessageHeader = ({ message }: { message: Message }) => {
  const name = message.author_name || "Bot";
  return (
    <View className="flex-row items-center gap-2">
      <BotAvatar src={message.author_avatar_url} name={name} />
      <Text numberOfLines={1} className="shrink text-sm font-semibold">
        {name}
      </Text>
      <View className="rounded-md border border-border px-1">
        <Text className="text-[9px] font-medium uppercase tracking-wide">Bot</Text>
      </View>
      <Text className="font-mono text-xs text-muted-foreground"><RelativeTime iso={message.created_at} /></Text>
      {message.via && <Text numberOfLines={1} className="shrink text-xs text-muted-foreground">via {message.via}</Text>}
    </View>
  );
};
