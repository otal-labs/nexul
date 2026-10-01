import type { DocListItem, DocSortField } from "@/models/Doc";

export interface DocGroup {
  label: string;
  docs: DocListItem[];
}

// Newest first by the chosen timestamp, bucketed by the viewer's own calendar day on that same timestamp; empty buckets are dropped.
export const groupDocsByDay = (docs: DocListItem[], sortBy: DocSortField, now: Date = new Date()): DocGroup[] => {
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1).getTime();
  const groups: DocGroup[] = [
    { label: "Today", docs: [] },
    { label: "Yesterday", docs: [] },
    { label: "Earlier", docs: [] },
  ];
  const newestFirst = [...docs].sort((a, b) => Date.parse(b[sortBy]) - Date.parse(a[sortBy]));
  for (const doc of newestFirst) {
    const at = Date.parse(doc[sortBy]);
    const bucket = at >= today ? 0 : at >= yesterday ? 1 : 2;
    groups[bucket]?.docs.push(doc);
  }
  return groups.filter((group) => group.docs.length > 0);
};
