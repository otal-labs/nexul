import { DocAuthorAvatar } from "@/components/doc/DocAuthorAvatar";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { RowActions } from "@/components/listpane/RowActions";
import { useDocRowActions } from "@/hooks/useDocRowActions";
import type { DocListItem } from "@/models/Doc";
import { docPath } from "@/models/Project";

interface DocListRowProps {
  doc: DocListItem;
  projectToken: string;
  selected: boolean;
}

export const DocListRow = ({ doc, projectToken, selected }: DocListRowProps) => {
  const { onClone, onDelete } = useDocRowActions(doc, selected);
  return (
    <ListPaneRow
      to={docPath(projectToken, doc.id)}
      title={doc.title}
      snippet={doc.snippet ?? ""}
      selected={selected}
      meta={
        <>
          <span className="font-mono text-[11px] text-muted-foreground tabular-nums">{formatUpdatedAgo(doc.updated_at)}</span>
          {doc.created_by && <DocAuthorAvatar userId={doc.created_by} />}
        </>
      }
      actions={(onClone || onDelete) && <RowActions itemLabel={doc.title} onClone={onClone} onDelete={onDelete} />}
    />
  );
};
