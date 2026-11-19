import { CheckIcon } from "lucide-react";
import { useState } from "react";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { menuItemClass } from "@/components/MenuItem";
import { useFetchDocFolders, useMoveDoc } from "@/hooks/DocFolderHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { Doc } from "@/models/Doc";
import { folderOptions } from "@/utils/DocFolderTreeUtility";

// The doc page's Move to folder row unfolds the project's folders in place, each named by its path, since the actions popover has no submenus.
export const DocMoveToFolderItem = ({ doc }: { doc: Doc }) => {
  const answer = useEntityActions("doc", doc.id);
  const { data: folders } = useFetchDocFolders(doc.project_id);
  const moveDoc = useMoveDoc();
  const [open, setOpen] = useState(false);
  if (!folders || folders.length < 2) return null;

  return (
    <>
      <ActionMenuItem
        permission="docs:write"
        answer={answer("docs:write")}
        verb="move docs"
        onSelect={() => setOpen((was) => !was)}
      >
        Move to folder
      </ActionMenuItem>
      {open && (
        <div role="group" aria-label="Folders" className="flex flex-col gap-0.5 pl-3">
          {folderOptions(folders).map((folder) => (
            <button
              key={folder.id}
              type="button"
              aria-current={folder.id === doc.folder_id}
              className={menuItemClass}
              onClick={() => {
                setOpen(false);
                if (folder.id !== doc.folder_id) moveDoc.mutate({ id: doc.id, folderId: folder.id });
              }}
            >
              <span className="flex-1 truncate">{folder.name}</span>
              {folder.id === doc.folder_id && <CheckIcon aria-hidden />}
            </button>
          ))}
        </div>
      )}
    </>
  );
};
