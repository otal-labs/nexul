import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { attachmentPath, type Attachment } from "@/models/Attachment";

export const getAttachmentsKey = "getAttachments";
export const getNoteMarkdownKey = "getNoteMarkdown";

export const useFetchConversationFiles = (conversationId: string | undefined) =>
  useQuery({
    queryKey: [getAttachmentsKey, conversationId],
    queryFn: () => api.get<Attachment[]>(`/api/attachments?conversation_id=${encodeURIComponent(conversationId ?? "")}`),
    enabled: !!conversationId,
  });

// The file route answers text/markdown, which the client hands back as the raw string.
export const useFetchNoteMarkdown = (attachmentId: string | undefined) =>
  useQuery({
    queryKey: [getNoteMarkdownKey, attachmentId],
    queryFn: async () => String(await api.get<unknown>(attachmentPath(attachmentId ?? ""))),
    enabled: !!attachmentId,
  });
