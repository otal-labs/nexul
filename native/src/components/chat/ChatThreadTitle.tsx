import { Stack } from "expo-router";
import type { ReactNode } from "react";

import { conversationLabel } from "@nexul/client-core/chat";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useFetchDoc } from "@/hooks/DocHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { ticketKey } from "@/models/Ticket";

interface ChatThreadTitleProps {
  conversationId: string;
  children: ReactNode;
}

// Reads the list cache, so a thread opened from the list or an Inbox link gets the same title, and so do its calendar events.
export const ChatThreadTitle = ({ conversationId, children }: ChatThreadTitleProps) => {
  const workspaceId = useCurrentWorkspaceId();
  const { data: conversations } = useFetchConversations(workspaceId);
  const { data: me } = useFetchMe(true);
  const resolvePerson = usePersonLookup(workspaceId);
  const conversation = conversations?.find((c) => c.id === conversationId);
  const { data: ticket } = useFetchTicket(conversation?.ticket_id);
  const { data: project } = useFetchProject(ticket?.project_id);
  const { data: doc } = useFetchDoc(conversation?.kind === "doc_thread" ? conversation.doc_id : undefined);
  const label = conversation ? conversationLabel(conversation, { currentUserId: me?.user.id, resolvePerson }) : "";
  const title = ticket ? `Thread · ${ticketKey(ticket, project?.prefix)}` : label;
  return (
    <CalendarTitleContext value={doc?.title ?? ticket?.title ?? (label || "Nexul")}>
      <Stack.Screen options={{ title }} />
      {children}
    </CalendarTitleContext>
  );
};
