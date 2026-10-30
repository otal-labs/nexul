import { DocStateIcon } from "@/components/doc/DocStateIcon";
import { formatShortAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { useDocRunState } from "@/hooks/TrailHooks";
import { useDocRowActions } from "@/hooks/useDocRowActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useDocSortStore } from "@/stores/docSortStore";
import type { DocListItem } from "@/models/Doc";
import { docPath } from "@/models/Project";
import { formatFullTime } from "@/utils/TimeUtility";

interface DocListRowProps {
  doc: DocListItem;
  projectToken: string;
  selected: boolean;
}

// One line: the state slot, the title, then the lock and the time the list is sorted by.
export const DocListRow = ({ doc, projectToken, selected }: DocListRowProps) => {
  const menu = useDocRowActions(doc, selected);
  const wsPath = useWorkspacePath();
  const runState = useDocRunState(doc.project_id, doc.id);
  const sortBy = useDocSortStore((s) => s.sortBy);
  return (
    <ListPaneRow
      to={wsPath(docPath(projectToken, doc.id))}
      title={doc.title}
      dense
      leading={<DocStateIcon runState={runState} locked={!!doc.locked} questionsClosed={!!doc.clarification_closed} className="ml-5" />}
      meta={
        <span title={`${sortBy === "created_at" ? "Created" : "Edited"} ${formatFullTime(doc[sortBy])}`} className="font-mono text-[11px] text-muted-foreground tabular-nums">
          {formatShortAgo(doc[sortBy])}
        </span>
      }
      selected={selected}
      menu={menu}
    />
  );
};
