import type { DocListItem } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import { subtreeIds } from "@/utils/DocFolderTreeUtility";

/** What a drag in the docs list holds, carried as the draggable's data. */
export type DocDragItem = { kind: "doc"; doc: DocListItem } | { kind: "folder"; folder: DocFolder };

export type DocDrop = { kind: "doc"; id: string; folderId: string } | { kind: "folder"; id: string; parentId: string };

// The move a drop onto a folder makes, or null where it would change nothing or put a folder inside itself.
export const docDrop = (item: DocDragItem | undefined, target: DocFolder | undefined, folders: DocFolder[]): DocDrop | null => {
  if (!item || !target) return null;
  if (item.kind === "doc") return item.doc.folder_id === target.id ? null : { kind: "doc", id: item.doc.id, folderId: target.id };
  if (item.folder.parent_id === target.id || subtreeIds(folders, item.folder.id).has(target.id)) return null;
  return { kind: "folder", id: item.folder.id, parentId: target.id };
};
