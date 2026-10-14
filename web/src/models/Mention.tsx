import type { MentionType } from "@nexul/client-core/mention";

export type MentionSearchType = MentionType | "person";

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
