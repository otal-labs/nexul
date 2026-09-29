import type { DocListItem } from "@/models/Doc";

export interface DocGroup {
  label: string;
  docs: DocListItem[];
}

// Newest first, bucketed by the viewer's own calendar day; empty buckets are dropped.
export const groupDocsByDay = (docs: DocListItem[], now: Date = new Date()): DocGroup[] => {
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1).getTime();
  const groups: DocGroup[] = [
    { label: "Today", docs: [] },
    { label: "Yesterday", docs: [] },
    { label: "Earlier", docs: [] },
  ];
  const newestFirst = [...docs].sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at));
  for (const doc of newestFirst) {
    const at = Date.parse(doc.updated_at);
    const bucket = at >= today ? 0 : at >= yesterday ? 1 : 2;
    groups[bucket]?.docs.push(doc);
  }
  return groups.filter((group) => group.docs.length > 0);
};
