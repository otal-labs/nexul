export type SearchKind = "ticket" | "doc" | "pull_request" | "stack" | "automation" | "channel" | "memory";

// One hit of the workspace search; in_body marks a ticket or doc found only through its text.
export interface SearchHit {
  type: SearchKind;
  id: string;
  title: string;
  hint?: string;
  status_label?: string;
  project_id?: string;
  // The ticket a pull request opens.
  ticket_id?: string;
  in_body: boolean;
  updated_at: string;
}

export interface SearchFound {
  hits: SearchHit[];
  // Kinds that had more hits than the limit.
  more: SearchKind[];
}
