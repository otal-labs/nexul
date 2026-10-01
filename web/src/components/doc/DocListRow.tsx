import { FileTextIcon, LockIcon } from "lucide-react";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { RowActions } from "@/components/listpane/RowActions";
import { useDocRowActions } from "@/hooks/useDocRowActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { DocListItem } from "@/models/Doc";
import { docPath } from "@/models/Project";

interface DocListRowProps {
  doc: DocListItem;
  projectToken: string;
  selected: boolean;
}

export const DocListRow = ({ doc, projectToken, selected }: DocListRowProps) => {
  const { pin, lock, moveTo, onClone, onDelete } = useDocRowActions(doc, selected);
  const wsPath = useWorkspacePath();
  return (
    <ListPaneRow
      to={wsPath(docPath(projectToken, doc.id))}
      title={doc.title}
      leading={<FileTextIcon className="ml-4.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
      titleIcon={doc.locked && <LockIcon className="size-3 shrink-0 text-muted-foreground" aria-label="Locked" />}
      selected={selected}
      actions={<RowActions itemLabel={doc.title} pin={pin} lock={lock} moveTo={moveTo} onClone={onClone} onDelete={onDelete} />}
    />
  );
};
