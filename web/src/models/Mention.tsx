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
  // A ticket's PREFIX-NUMBER.
  key?: string;
  // Tickets and docs: the title holds every word, or only the text does; a text match says where its words are.
  match?: "title" | "text";
  snippet?: string;
  heading?: string;
  note_id?: string;
  // People only.
  login?: string;
  avatar_url?: string;
}
