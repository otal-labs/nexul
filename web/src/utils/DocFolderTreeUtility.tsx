import type { DocFolder } from "@/models/DocFolder";

export interface DocFolderNode {
  folder: DocFolder;
  /** 0 for a top-level folder. */
  depth: number;
}

// A folder whose parent isn't listed (hidden from the viewer, or the list lagging a move) sits at the top.
const parentOf = (folders: DocFolder[], folder: DocFolder) => (folders.some((f) => f.id === folder.parent_id) ? folder.parent_id : "");

/** Folders depth first, each one's folders right after it, siblings in the server's order. */
export const folderTree = (folders: DocFolder[]): DocFolderNode[] => {
  const out: DocFolderNode[] = [];
  const seen = new Set<string>();
  const walk = (parentId: string, depth: number) => {
    for (const folder of folders) {
      if (seen.has(folder.id) || parentOf(folders, folder) !== parentId) continue;
      seen.add(folder.id);
      out.push({ folder, depth });
      walk(folder.id, depth + 1);
    }
  };
  walk("", 0);
  return out;
};

/** The folder and those it sits inside, outermost first. */
export const folderChain = (folders: DocFolder[], id: string): DocFolder[] => {
  const chain: DocFolder[] = [];
  for (let folder = folders.find((f) => f.id === id); folder && !chain.includes(folder); folder = folders.find((f) => f.id === folder?.parent_id)) {
    chain.unshift(folder);
  }
  return chain;
};

/** "Clients / Acme": the names from the top down to the folder. */
export const folderPath = (folders: DocFolder[], id: string) =>
  folderChain(folders, id)
    .map((f) => f.name)
    .join(" / ");

/** The ids of the folder and every folder inside it. */
export const subtreeIds = (folders: DocFolder[], id: string): Set<string> => {
  const ids = new Set([id]);
  for (const { folder } of folderTree(folders)) {
    if (ids.has(folder.parent_id)) ids.add(folder.id);
  }
  return ids;
};

/** Every folder as a pick, in tree order and named by its path. */
export const folderOptions = (folders: DocFolder[]) => folderTree(folders).map(({ folder }) => ({ id: folder.id, name: folderPath(folders, folder.id) }));
