// Mirrors internal/mentions/find.go: what GET /api/search returns.
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

export interface SearchGroup {
  heading: string;
  hits: SearchHit[];
}

// The server's kind order.
export const searchKinds: { kind: SearchKind; heading: string }[] = [
  { kind: "ticket", heading: "Tickets" },
  { kind: "doc", heading: "Docs" },
  { kind: "pull_request", heading: "Pull requests" },
  { kind: "stack", heading: "Stacks" },
  { kind: "automation", heading: "Automations" },
  { kind: "channel", heading: "Channels" },
  { kind: "memory", heading: "Memories" },
];

// A group per kind of title hits, then the tickets and docs found only in their text under In text; only the kinds
// shown count, so an app that cannot open a kind leaves it out. Empty groups are dropped.
export const groupSearchHits = (hits: SearchHit[], shown: SearchKind[] = searchKinds.map((k) => k.kind)): SearchGroup[] => {
  const kept = hits.filter((hit) => shown.includes(hit.type));
  const titled = searchKinds.map(({ kind, heading }) => ({ heading, hits: kept.filter((hit) => hit.type === kind && !hit.in_body) }));
  return [...titled, { heading: "In text", hits: kept.filter((hit) => hit.in_body) }].filter((group) => group.hits.length > 0);
};

// Whether a shown kind had more hits than the answer holds, so the app can suggest another word.
export const searchCut = (found: SearchFound | undefined, shown: SearchKind[] = searchKinds.map((k) => k.kind)): boolean =>
  !!found && found.hits.length > 0 && found.more.some((kind) => shown.includes(kind));
