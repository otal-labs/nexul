import type { DocListItem } from "@/models/Doc";
import type { TrailState } from "@/models/Trail";
import { daysAgo } from "@/utils/TimeUtility";

export interface AttentionDoc {
  doc: DocListItem;
  state: TrailState;
}

export interface DocsOverviewLists {
  /** Waiting docs first, then running, newest edit first within each; at most five. */
  attention: AttentionDoc[];
  /** The newest edits; at most twenty. */
  recent: DocListItem[];
}

const newestEdit = (a: DocListItem, b: DocListItem) => Date.parse(b.updated_at) - Date.parse(a.updated_at);

// `docs` are the live docs the viewer can open, `states` their runs in the same order.
export const overviewLists = (docs: DocListItem[], states: (TrailState | undefined)[]): DocsOverviewLists => {
  const withState = docs.flatMap((doc, i) => {
    const state = states[i];
    return state ? [{ doc, state }] : [];
  });
  const rank = (a: AttentionDoc) => (a.state === "waiting" ? 0 : 1);
  const attention = withState.sort((a, b) => rank(a) - rank(b) || newestEdit(a.doc, b.doc)).slice(0, 5);
  return { attention, recent: [...docs].sort(newestEdit).slice(0, 20) };
};

export const docMatches = (query: string) => (doc: DocListItem) =>
  doc.title.toLowerCase().includes(query) || (doc.snippet ?? "").toLowerCase().includes(query);

export interface DayGroup {
  label: string;
  docs: DocListItem[];
}

// Newest-first docs split at today and the last week; empty groups are left out.
export const dayGroups = (docs: DocListItem[]): DayGroup[] =>
  [
    { label: "Today", docs: docs.filter((d) => daysAgo(d.updated_at) < 1) },
    { label: "Last 7 days", docs: docs.filter((d) => daysAgo(d.updated_at) >= 1 && daysAgo(d.updated_at) < 7) },
    { label: "Earlier", docs: docs.filter((d) => daysAgo(d.updated_at) >= 7) },
  ].filter((group) => group.docs.length > 0);
