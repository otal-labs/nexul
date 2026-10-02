// Mirrors the server's attachment list item; the phone only reads a note's file.
export interface Attachment {
  id: string;
  conversation_id?: string;
  name: string;
  content_type: string;
  size: number;
  created_at: string;
}

export const attachmentPath = (id: string): string => `/api/attachments/${id}`;

export const formatBytes = (size: number): string => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(0)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};
