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

// A pasted screenshot arrives with no file name.
export const uploadName = (file: File): string => file.name || "Pasted image.png";

// Mirrors the server's Inline(): raster images render in place, SVG and everything else download.
export const isInlineImage = (contentType: string): boolean =>
  contentType.startsWith("image/") && !contentType.includes("svg");

export type PreviewKind = "image" | "svg" | "pdf" | "audio" | "csv" | "too-large" | "none";

// The upload limit; a file past it, from before a raise, downloads instead of loading into the page.
export const previewMaxBytes = 10 * 1024 * 1024;

const audioExtensions = new Set(["mp3", "wav", "ogg", "oga", "opus", "m4a", "aac", "flac", "weba"]);

const extensionOf = (name: string): string => {
  const dot = name.lastIndexOf(".");
  return dot < 0 ? "" : name.slice(dot + 1).toLowerCase();
};

// The server sniffs types from the bytes, so SVG and CSV arrive as text and some audio as video: the name picks, the sniff guards.
const kindOf = (ext: string, contentType: string): PreviewKind => {
  const text = contentType.startsWith("text/");
  if (isInlineImage(contentType)) return "image";
  if (ext === "svg" && (text || contentType.includes("svg"))) return "svg";
  if (contentType === "application/pdf") return "pdf";
  if (contentType.startsWith("audio/") || (audioExtensions.has(ext) && !text)) return "audio";
  if ((ext === "csv" || ext === "tsv") && text) return "csv";
  return "none";
};

export const previewKind = ({ name, content_type, size }: Pick<Attachment, "name" | "content_type" | "size">): PreviewKind => {
  const kind = kindOf(extensionOf(name), content_type);
  if (kind !== "none" && size > previewMaxBytes) return "too-large";
  return kind;
};

export const formatBytes =(size: number): string => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(0)} KB`;
  if (size < 1024 ** 3) return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  return `${(size / 1024 ** 3).toFixed(1)} GB`;
};
