import { useRouter } from "expo-router";
import { FileText, Hash, SquareKanban, User } from "lucide-react-native";
import { Pressable } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { useConversationLabel } from "@/hooks/ChatHooks";
import type { Conversation, ConversationKind, DMLabelContext } from "@/models/Chat";

const kindIcons: Partial<Record<ConversationKind, typeof Hash>> = {
  dm: User,
  ticket_thread: SquareKanban,
  doc_thread: FileText,
};

interface ConversationRowProps {
  conversation: Conversation;
  unreadCount: number;
  dmCtx: DMLabelContext;
}

export const ConversationRow = ({ conversation, unreadCount, dmCtx }: ConversationRowProps) => {
  const router = useRouter();
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  const label = useConversationLabel(conversation, dmCtx);
  const Icon = kindIcons[conversation.kind] ?? Hash;
  return (
    <Pressable
      role="button"
      aria-label={label}
      onPress={() => router.push({ pathname: "/chat/[id]", params: { id: conversation.id } })}
      className="min-h-12 flex-row items-center gap-3 border-b border-border px-4 py-3 active:bg-accent"
    >
      <Icon color={String(mutedForeground)} size={16} />
      <Text numberOfLines={1} className={cn("flex-1", unreadCount > 0 && "font-semibold")}>
        {label}
      </Text>
      {unreadCount > 0 && (
        <Text aria-label={`${unreadCount} unread`} className="font-mono text-xs text-muted-foreground">
          {unreadCount}
        </Text>
      )}
    </Pressable>
  );
};
