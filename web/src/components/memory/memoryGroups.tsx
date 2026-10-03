import type { Memory } from "@/models/Memory";

export type MemoryFolderId = "main" | "footer";

export interface MemoryFolder {
  id: MemoryFolderId;
  label: string;
  memories: Memory[];
}

export const memoryFolderOptions: { id: MemoryFolderId; name: string }[] = [
  { id: "main", name: "Main" },
  { id: "footer", name: "Footer" },
];

const requiredFirst = (a: Memory, b: Memory) =>
  Number(b.always_included) - Number(a.always_included) || Date.parse(b.updated_at) - Date.parse(a.updated_at);

// Main holds what a run reads first, Footer what it concludes with; required memories lead each, then newest first.
export const groupMemoriesByFolder = (memories: Memory[]): MemoryFolder[] => {
  const sorted = [...memories].sort(requiredFirst);
  return memoryFolderOptions.map(({ id, name }) => ({
    id,
    label: name,
    memories: sorted.filter((m) => m.footer === (id === "footer")),
  }));
};
