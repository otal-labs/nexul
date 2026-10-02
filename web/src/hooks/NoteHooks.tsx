import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchTicket } from "@/hooks/TicketHooks";
import { attachmentPath } from "@/models/Attachment";

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
