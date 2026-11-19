import type { DocListItem, DocSortField } from "@/models/Doc";
import type { DocFolder } from "@/models/DocFolder";
import { folderTree } from "@/utils/DocFolderTreeUtility";

export interface DocGroup {
  label: string;
  docs: DocListItem[];
}

export interface DocFolderGroup {
  folder: DocFolder;
  /** The folder's listed rows: matching, not pinned, newest first. */
  docs: DocListItem[];
  /** Every doc the viewer can open in the folder and the folders inside it, pinned and unmatched ones included. */
  total: number;
  /** The folders inside it, in the server's order. */
  folders: DocFolderGroup[];
}

export interface DocListGroups {
  pinned: DocListItem[];
  /** The top-level folders, each holding the folders inside it. */
  folders: DocFolderGroup[];
}

interface GroupDocsInput {
  docs: DocListItem[];
  /** In the server's order: the default folder first, then creation order. */
  folders: DocFolder[];
  sortBy: DocSortField;
  pinnedIds: string[];
  /** Set while a search runs; folders with nothing matching, in them or inside them, are left out. */
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
  const tree = folderTree(folders);
  const build = (parentId: string, depth: number): DocFolderGroup[] =>
    tree
      .filter((node) => node.depth === depth && (depth === 0 || node.folder.parent_id === parentId))
      .map(({ folder }) => {
        const inside = build(folder.id, depth + 1);
        const own = docs.filter((doc) => folderOf(doc) === folder.id).length;
        return {
          folder,
          docs: newestFirst.filter((doc) => folderOf(doc) === folder.id),
          total: own + inside.reduce((sum, g) => sum + g.total, 0),
          folders: match ? inside.filter(holdsAMatch) : inside,
        };
      });
  const groups = build("", 0);
  if (!match) return { pinned, folders: groups };
  return { pinned, folders: groups.filter(holdsAMatch) };
};

const holdsAMatch = (g: DocFolderGroup): boolean => g.docs.length > 0 || g.folders.length > 0;
