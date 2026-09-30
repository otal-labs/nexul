// What a chip resolves through /api/mentions/resolve; people resolve from the workspace's People instead.
export type MentionType = "ticket" | "doc";

export type MentionSearchType = MentionType | "person";

export interface MentionRef {
  type: MentionType;
  id: string;
}

export interface MentionChipData {
  type: MentionType;
  id: string;
  title: string;
  status?: string;
  status_label?: string;
  can_open: boolean;
  // Back the mention-chip layout template's tokens; ticket-only, doc chips leave them undefined.
  project_prefix?: string;
  project_number?: number;
  type_label?: string;
  developer_label?: string;
  due_label?: string;
}

export interface MentionSearchResult {
  type: MentionSearchType;
  // A person's user id.
  id: string;
  // A person's display name.
  title: string;
  status_label?: string;
  can_open: boolean;
  // People only.
  login?: string;
  avatar_url?: string;
}
