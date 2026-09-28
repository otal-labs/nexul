export interface DocListItem {
  id: string;
  project_id: string;
  title: string;
  version: number;
  archived: boolean;
  can_open: boolean;
  updated_at: string;
}

export interface Doc {
  id: string;
  project_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface RichMark {
  type: string;
  attrs?: Record<string, unknown>;
}

// Doc.body is canonical Tiptap JSON (ADR 0026): a {"type":"doc",...} object or a bare top-level array of blocks.
export interface RichNode {
  type: string;
  text?: string;
  marks?: RichMark[];
  attrs?: Record<string, unknown>;
  content?: RichNode[];
}

interface RichDoc {
  type: string;
  content?: RichNode[];
}

// Mirrors richtext.UnmarshalJSON; a non-JSON legacy body (none in production data today, per no-backfill policy)
// returns null so the caller falls back to plain text.
export const parseRichBody = (body: string): RichNode[] | null => {
  try {
    const parsed = JSON.parse(body) as unknown;
    if (Array.isArray(parsed)) return parsed as RichNode[];
    if (typeof parsed === "object" && parsed !== null && (parsed as RichDoc).type === "doc") {
      return (parsed as RichDoc).content ?? [];
    }
    return null;
  } catch {
    return null;
  }
};
