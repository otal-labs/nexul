import type { Memory } from "@/models/Memory";

export interface MemoryGroup {
  label: string;
  memories: Memory[];
}

// Always-included memories lead as Pinned, since they reach every turn; each group is newest first.
export const groupMemoriesByPin = (memories: Memory[]): MemoryGroup[] => {
  const newestFirst = [...memories].sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at));
  return [
    { label: "Pinned", memories: newestFirst.filter((m) => m.always_included) },
    { label: "Other", memories: newestFirst.filter((m) => !m.always_included) },
  ].filter((group) => group.memories.length > 0);
};
