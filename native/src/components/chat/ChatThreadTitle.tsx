import { Stack } from "expo-router";

import { conversationLabel } from "@nexul/client-core/chat";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { ticketKey } from "@/models/Ticket";

interface ChatThreadTitleProps {
  conversationId: string;
}

// Reads the list cache, so a thread opened from the list or from an Inbox link gets the same title.
export const ChatThreadTitle = ({ conversationId }: ChatThreadTitleProps) => {
  const workspaceId = useCurrentWorkspaceId();
  const { data: conversations } = useFetchConversations(workspaceId);
  const { data: me } = useFetchMe(true);
  const resolvePerson = usePersonLookup(workspaceId);
  const conversation = conversations?.find((c) => c.id === conversationId);
  const { data: ticket } = useFetchTicket(conversation?.ticket_id);
  const { data: project } = useFetchProject(ticket?.project_id);
  const label = conversation ? conversationLabel(conversation, { currentUserId: me?.user.id, resolvePerson }) : "";
  const title = ticket ? `Thread · ${ticketKey(ticket, project?.prefix)}` : label;
  return <Stack.Screen options={{ title }} />;
};
