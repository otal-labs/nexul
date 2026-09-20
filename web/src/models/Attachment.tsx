export interface Attachment {
  id: string;
  doc_id?: string;
  ticket_id?: string;
  conversation_id?: string;
  memory_id?: string;
  name: string;
  content_type: string;
  size: number;
  uploaded_by: string;
  created_at: string;
}

// The doc, ticket, conversation, or memory a file hangs off (exactly one); doubles as the list query's params.
export type AttachmentOwner =
  | { doc_id: string }
  | { ticket_id: string }
  | { conversation_id: string }
  | { memory_id: string };

export const attachmentPath = (id: string): string => `/api/attachments/${id}`;

export const isAttachmentPath = (src: string): boolean => src.startsWith("/api/attachments/");

// Mirrors the server's Inline(): raster images render in place, SVG and everything else download.
export const isInlineImage = (contentType: string): boolean =>
  contentType.startsWith("image/") && !contentType.includes("svg");

export const formatBytes = (size: number): string => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(0)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};
