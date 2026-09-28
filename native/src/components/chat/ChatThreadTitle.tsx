import { Stack } from "expo-router";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useChatAuthorLookup, useChatWorkspaceId, useFetchConversations } from "@/hooks/ChatHooks";
import { conversationLabel } from "@/models/Chat";

interface ChatThreadTitleProps {
  conversationId: string;
}

// Reads the list cache, so a thread opened from the list or from an Inbox link gets the same title.
export const ChatThreadTitle = ({ conversationId }: ChatThreadTitleProps) => {
  const workspaceId = useChatWorkspaceId();
  const { data: conversations } = useFetchConversations(workspaceId);
  const { data: me } = useFetchMe(true);
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const conversation = conversations?.find((c) => c.id === conversationId);
  const title = conversation ? conversationLabel(conversation, { currentUserId: me?.user.id, resolveLogin }) : "";
  return <Stack.Screen options={{ title }} />;
};
