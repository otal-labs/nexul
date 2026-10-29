import type { Memory } from "@/models/Memory";

export type MemoryFolderId = "main" | "footer";

export interface MemoryFolder {
  id: MemoryFolderId;
  label: string;
  /** When a run reads the folder. */
  hint: string;
  memories: Memory[];
}

export const memoryFolderOptions: { id: MemoryFolderId; name: string; hint: string }[] = [
  { id: "main", name: "Main", hint: "Read when a run starts" },
  { id: "footer", name: "Footer", hint: "Read when a run ends" },
];

const requiredFirst = (a: Memory, b: Memory) =>
  Number(b.always_included) - Number(a.always_included) || Date.parse(b.updated_at) - Date.parse(a.updated_at);

// Main holds what a run reads first, Footer what it concludes with; required memories lead each, then newest first.
export const groupMemoriesByFolder = (memories: Memory[]): MemoryFolder[] => {
  const sorted = [...memories].sort(requiredFirst);
  return memoryFolderOptions.map(({ id, name, hint }) => ({
    id,
    label: name,
    hint,
    memories: sorted.filter((m) => m.footer === (id === "footer")),
  }));
};
