import { useQuery, type QueryClient } from "@tanstack/react-query";

import { isNote, type Conversation } from "@nexul/client-core/chat";

import { api } from "@/api/client";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { attachmentPath } from "@/models/Attachment";
import type { Message, TicketNotes } from "@/models/Chat";
import type { LiveFollower } from "@/lib/live";

export const getNoteTextKey = "getNoteText";
export const getTicketNotesKey = "getTicketNotes";

// A ticket's notes, newest first, with the thread they live in; the ticket page's Notes tab shows only when there are some.
export const useFetchTicketNotes = (ticketId: string | undefined) =>
  useQuery({
    queryKey: [getTicketNotesKey, ticketId],
    queryFn: async () => (await api.get<TicketNotes>(`/api/chat/tickets/${ticketId}/notes`)).data,
    enabled: !!ticketId,
  });

// A note's markdown through the file route, which serves it no-cache since it is edited in place.
export const useFetchNoteText = (attachmentId: string | undefined) =>
  useQuery({
    queryKey: [getNoteTextKey, attachmentId],
    queryFn: async () => (await api.get<string>(attachmentPath(attachmentId ?? ""), { responseType: "text" })).data,
    enabled: !!attachmentId,
  });

// Editing and deleting a note take tickets:write on its ticket's project, which is what the server checks.
export const useCanEditNote = (ticketId: string | undefined): boolean => {
  const { data: ticket } = useFetchTicket(ticketId);
  const can = useAreaAccess(ticket?.project_id);
  return !!ticket && (can?.("editNotes") ?? false);
};

// Patches every cached notes list of the conversation, which a ticket's list names beside its notes.
const patchNotes = (client: QueryClient, conversationId: string, patch: (notes: Message[]) => Message[]) =>
  client.setQueriesData<TicketNotes>(
    { queryKey: [getTicketNotesKey], predicate: (q) => (q.state.data as TicketNotes | undefined)?.conversation_id === conversationId },
    (list) => list && { ...list, notes: patch(list.notes) },
  );

export const noteFollower: LiveFollower = {
  // A ticket's first thread gives its notes list a conversation to be patched by.
  "chat.conversation.created": ({ conversation }: { conversation: Conversation }, { client }) =>
    !!conversation.ticket_id && client.invalidateQueries({ queryKey: [getTicketNotesKey, conversation.ticket_id], exact: true }),
  // A new note's file is one more of its thread's files, which its pill and its row are named from.
  "chat.message.created": ({ message }: { message: Message }, { client }) => {
    if (!isNote(message)) return undefined;
    patchNotes(client, message.conversation_id, (notes) => [message, ...notes.filter((n) => n.id !== message.id)]);
    return client.invalidateQueries({ queryKey: [getAttachmentsKey, { conversation_id: message.conversation_id }], exact: true });
  },
  // A note's file changed under its message: open renders and the pill's size follow it; an open editor follows its room.
  "chat.message.updated": ({ message }: { message: Message }, { client }) => {
    if (!isNote(message)) return undefined;
    patchNotes(client, message.conversation_id, (notes) => notes.map((n) => (n.id === message.id ? message : n)));
    return Promise.all([
      client.invalidateQueries({ queryKey: [getNoteTextKey, message.attachment_id], exact: true }),
      client.invalidateQueries({ queryKey: [getAttachmentsKey, { conversation_id: message.conversation_id }], exact: true }),
    ]);
  },
  "chat.message.deleted": ({ conversation_id: conversationId, message_id: messageId }: { conversation_id: string; message_id: string }, { client }) => {
    patchNotes(client, conversationId, (notes) => notes.filter((n) => n.id !== messageId));
  },
};
