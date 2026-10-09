import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import { defineQuery } from "@/lib/liveQuery";
import { attachmentPath, type Attachment } from "@/models/Attachment";

export const getAttachmentsKey = "getAttachments";
export const getNoteMarkdownKey = "getNoteMarkdown";

const conversationFilesQuery = defineQuery({
  key: getAttachmentsKey,
  fetch: (conversationId: string | undefined) =>
    api.get<Attachment[]>(`/api/attachments?conversation_id=${encodeURIComponent(conversationId ?? "")}`),
  refreshes: {},
});

export const useFetchConversationFiles = (conversationId: string | undefined) =>
  useQuery({ ...conversationFilesQuery.options(conversationId), enabled: !!conversationId });

// The file route answers text/markdown, which the client hands back as the raw string.
const noteMarkdownQuery = defineQuery({
  key: getNoteMarkdownKey,
  fetch: async (attachmentId: string | undefined) => String(await api.get<unknown>(attachmentPath(attachmentId ?? ""))),
  refreshes: {},
});

export const useFetchNoteMarkdown = (attachmentId: string | undefined) =>
  useQuery({ ...noteMarkdownQuery.options(attachmentId), enabled: !!attachmentId });
