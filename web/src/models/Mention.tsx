export type MentionType = "ticket" | "doc";

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
  type: MentionType;
  id: string;
  title: string;
  status_label?: string;
  can_open: boolean;
}
