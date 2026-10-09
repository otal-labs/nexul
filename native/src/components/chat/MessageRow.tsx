import { View } from "react-native";

import { dropEchoedAuthor } from "@nexul/client-core/embed";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedStack } from "@/components/chat/EmbedStack";
import { HandoffPill } from "@/components/chat/HandoffPill";
import { MessageAuthorLine, MessageAvatar } from "@/components/chat/MessageAuthor";
import { MessageBody } from "@/components/chat/MessageBody";
import { MessageReactions } from "@/components/chat/MessageReactions";
import { NoteFilePill } from "@/components/chat/NoteFilePill";
import { Text } from "@/components/ui/text";
import { BotMessageIdContext } from "@/hooks/BotMediaHooks";
import { cn } from "@/lib/utils";
import { isNote, type Message } from "@/models/Chat";

interface MessageRowProps {
  message: Message;
  // Your own message is the one ember bubble, on the right with no header.
  own: boolean;
  // The previous message is the same author's, so this one drops its avatar and header.
  continuation?: boolean;
}

const OwnMessage = ({ message, continuation }: { message: Message; continuation: boolean }) => (
  <View className={cn("items-end gap-1 px-4", continuation ? "pt-1" : "pt-3", message.pending && "opacity-70")}>
    <View className="max-w-[82%] rounded-xl rounded-br-sm bg-brand px-3.5 py-2">
      <MessageBody body={message.body} own />
    </View>
    {message.reactions && message.reactions.length > 0 && <MessageReactions reactions={message.reactions} />}
  </View>
);

// Everyone else's text runs plain under a 32pt avatar, so a long thread reads as a transcript, not a stack of boxes.
const OtherMessage = ({ message, continuation }: { message: Message; continuation: boolean }) => {
  const isBot = message.author_kind === "bot";
  const embeds = isBot ? (message.embeds ?? []).map((embed) => dropEchoedAuthor(embed, message.author_name || "Bot")) : [];
  return (
    <View className={cn("flex-row gap-3 px-4", continuation ? "pt-1" : "pt-4")}>
      <View className="w-8">{!continuation && <MessageAvatar message={message} />}</View>
      <View className="min-w-0 flex-1 gap-1">
        {!continuation && <MessageAuthorLine message={message} />}
        {!isBot && <MessageBody body={message.body} />}
        {isBot && message.body !== "" && <DiscordMarkdown text={message.body} mentionHandles={(message.mentions ?? []).map((m) => m.handle)} />}
        {embeds.length > 0 && <EmbedStack embeds={embeds} />}
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
    </View>
  );
};

export const MessageRow = ({ message, own, continuation = false }: MessageRowProps) => {
  const isSystem = message.author_kind === "system";
  return (
    // A bot is never resolved as a person: its id is not a member, so it shows the name it posted with.
    <BotMessageIdContext value={message.author_kind === "bot" ? message.id : undefined}>
      {isSystem && <Text className="px-4 pt-3 text-center text-xs italic text-muted-foreground">{message.body}</Text>}
      {!isSystem && own && <OwnMessage message={message} continuation={continuation} />}
      {!isSystem && !own && <OtherMessage message={message} continuation={continuation} />}
    </BotMessageIdContext>
  );
};
