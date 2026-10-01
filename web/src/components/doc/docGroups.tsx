import type { DocListItem, DocSortField } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";

export interface DocGroup {
  label: string;
  docs: DocListItem[];
}

export interface DocFolderGroup {
  folder: DocFolder;
  /** The folder's listed rows: matching, not pinned, newest first. */
  docs: DocListItem[];
  /** Every doc in the folder the viewer can open, pinned and unmatched ones included. */
  total: number;
}

export interface DocListGroups {
  pinned: DocListItem[];
  folders: DocFolderGroup[];
}

interface GroupDocsInput {
  docs: DocListItem[];
  /** In the server's order: the default folder first, then creation order. */
  folders: DocFolder[];
  sortBy: DocSortField;
  pinnedIds: string[];
  /** Set while a search runs; folders with nothing matching are left out. */
  match?: ((doc: DocListItem) => boolean) | undefined;
}

// A doc whose folder is not listed (the folder list lagging a move) shows under the default folder.
export const groupDocs = ({ docs, folders, sortBy, pinnedIds, match }: GroupDocsInput): DocListGroups => {
  const fallback = (folders.find((f) => f.is_default) ?? folders[0])?.id;
  const known = new Set(folders.map((f) => f.id));
  const folderOf = (doc: DocListItem) => (known.has(doc.folder_id) ? doc.folder_id : fallback);
  const shown = match ? docs.filter(match) : docs;
  const byId = new Map(shown.map((doc) => [doc.id, doc]));
  const pinned = pinnedIds.flatMap((id) => byId.get(id) ?? []);
  const pinnedSet = new Set(pinned.map((doc) => doc.id));
  const newestFirst = shown.filter((doc) => !pinnedSet.has(doc.id)).sort((a, b) => Date.parse(b[sortBy]) - Date.parse(a[sortBy]));
  const groups = folders.map((folder) => ({
    folder,
    docs: newestFirst.filter((doc) => folderOf(doc) === folder.id),
    total: docs.filter((doc) => folderOf(doc) === folder.id).length,
  }));
  if (!match) return { pinned, folders: groups };
  return { pinned, folders: groups.filter((g) => g.docs.length > 0) };
};
