import { useRouter } from "expo-router";
import { FileText, Hash, Lock, SquareKanban, Users } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { UnreadBadge } from "@/components/UnreadBadge";
import { useConversationLabel } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import type { Conversation, DMLabelContext } from "@/models/Chat";

interface ConversationRowProps {
  conversation: Conversation;
  unreadCount: number;
  dmCtx: DMLabelContext;
}

// A direct message with one other person leads with their avatar; everything else with a muted glyph on a tile.
const ConversationMark = ({ conversation, dmCtx }: Pick<ConversationRowProps, "conversation" | "dmCtx">) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const others = (conversation.participant_ids ?? []).filter((id) => id !== dmCtx.currentUserId);
  const other = conversation.kind === "dm" && others.length === 1 && others[0];
  const Icon =
    (conversation.private && Lock) ||
    (conversation.kind === "dm" && Users) ||
    (conversation.kind === "ticket_thread" && SquareKanban) ||
    (conversation.kind === "doc_thread" && FileText) ||
    Hash;
  return (
    <>
      {other && <PersonAvatar person={dmCtx.resolvePerson(other)} size={30} />}
      {!other && (
        <View className="size-[30px] items-center justify-center rounded-md border border-border bg-card">
          <Icon color={String(muted)} size={15} />
        </View>
      )}
    </>
  );
};

export const ConversationRow = ({ conversation, unreadCount, dmCtx }: ConversationRowProps) => {
  const router = useRouter();
  const label = useConversationLabel(conversation, dmCtx);
  const unread = unreadCount > 0;
  return (
    <Pressable
      role="button"
      aria-label={conversation.private ? `${label}, private` : label}
      onPress={() => router.push({ pathname: "/chat/[id]", params: { id: conversation.id } })}
      className="min-h-[52px] flex-row items-center gap-3.5 px-5 py-2.5 active:bg-accent"
    >
      <ConversationMark conversation={conversation} dmCtx={dmCtx} />
      <Text numberOfLines={1} className={cn("min-w-0 flex-1 text-[15px]", unread ? "font-medium" : "text-muted-foreground")}>
        {label}
      </Text>
      {unread && <UnreadBadge count={unreadCount} />}
    </Pressable>
  );
};
