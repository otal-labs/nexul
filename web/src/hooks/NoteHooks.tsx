import { useQuery } from "@tanstack/react-query";

import { isNote } from "@nexul/client-core/chat";

import { api } from "@/api/client";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { attachmentPath } from "@/models/Attachment";
import type { Message } from "@/models/Chat";
import type { LiveFollower } from "@/lib/live";

export const getNoteTextKey = "getNoteText";

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

// A note's file changed under its message: open renders and the pill's size follow it; an open editor follows its room.
export const noteFollower: LiveFollower = {
  "chat.message.updated": ({ message }: { message: Message }, { client }) =>
    isNote(message) &&
    Promise.all([
      client.invalidateQueries({ queryKey: [getNoteTextKey, message.attachment_id], exact: true }),
      client.invalidateQueries({ queryKey: [getAttachmentsKey, { conversation_id: message.conversation_id }], exact: true }),
    ]),
};
