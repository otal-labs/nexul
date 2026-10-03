import { CircleHelpIcon, FileTextIcon, LoaderCircleIcon, LockIcon } from "lucide-react";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { RowActions } from "@/components/listpane/RowActions";
import { useDocRunState } from "@/hooks/TrailHooks";
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
  const runState = useDocRunState(doc.project_id, doc.id);
  return (
    <ListPaneRow
      to={wsPath(docPath(projectToken, doc.id))}
      title={doc.title}
      leading={
        <>
          {runState === "waiting" && (
            <CircleHelpIcon className="ml-4.5 size-3.5 shrink-0 text-info" role="img" aria-label="Play waiting for an answer" />
          )}
          {runState !== undefined && runState !== "waiting" && (
            <LoaderCircleIcon className="ml-4.5 size-3.5 shrink-0 animate-spin text-warning motion-reduce:animate-none" role="img" aria-label="Play running" />
          )}
          {runState === undefined && <FileTextIcon className="ml-4.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
        </>
      }
      titleIcon={doc.locked && <LockIcon className="size-3 shrink-0 text-muted-foreground" aria-label="Locked" />}
      selected={selected}
      actions={<RowActions itemLabel={doc.title} pin={pin} lock={lock} moveTo={moveTo} onClone={onClone} onDelete={onDelete} />}
    />
  );
};
