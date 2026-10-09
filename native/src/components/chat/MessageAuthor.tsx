import { View } from "react-native";

import { personLabel } from "@nexul/client-core/person";

import { BotAvatar } from "@/components/chat/BotAvatar";
import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { formatMessageTime } from "@/lib/time";
import type { Message } from "@/models/Chat";

// The 32pt face beside a run of someone's messages: a person's avatar, a bot's picture, the Nexul glyph for the Agent.
export const MessageAvatar = ({ message }: { message: Message }) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  return (
    <>
      {message.author_kind === "user" && <PersonAvatar person={resolvePerson(message.author_id)} size={32} />}
      {message.author_kind === "bot" && <BotAvatar src={message.author_avatar_url} name={message.author_name || "Bot"} size={32} />}
      {message.author_kind === "agent" && <BotAvatar src={undefined} name="Agent" size={32} />}
    </>
  );
};

// The name in 13pt semibold and the mono clock time; a bot's post carries the BOT tag so it never passes for a person.
export const MessageAuthorLine = ({ message }: { message: Message }) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  const isBot = message.author_kind === "bot";
  const name = (isBot && (message.author_name || "Bot")) || (message.author_kind === "agent" && "Agent") || personLabel(resolvePerson(message.author_id));
  return (
    <View className="flex-row items-center gap-2">
      <Text numberOfLines={1} className="shrink text-[13px] font-semibold">
        {name}
      </Text>
      {isBot && (
        <View className="rounded-sm border border-input px-1">
          <Text className="font-mono text-[9px] font-medium uppercase tracking-[0.8px] text-muted-foreground">Bot</Text>
        </View>
      )}
      <Text className="font-mono text-[11px] text-muted-foreground">{formatMessageTime(message.created_at)}</Text>
      {message.via && <Text numberOfLines={1} className="shrink text-xs text-muted-foreground">via {message.via}</Text>}
      {message.edited_at && <Text className="text-xs text-muted-foreground">(edited)</Text>}
    </View>
  );
};
