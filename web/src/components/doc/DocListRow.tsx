import { useDraggable } from "@dnd-kit/core";
import { useId } from "react";

import type { DocDragItem } from "@/components/doc/docDrag";
import { DocStateIcon } from "@/components/doc/DocStateIcon";
import { formatShortAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { docQuery } from "@/hooks/DocHooks";
import { useDocRunState } from "@/hooks/TrailHooks";
import { useHoverPrefetch } from "@/hooks/useHoverPrefetch";
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
  // A pinned doc is listed twice, so each row has its own drag id.
  const dragId = useId();
  // Held only by a viewer who may move it, the same gate as the menu's Move to folder.
  const drag = useDraggable({ id: dragId, data: { kind: "doc", doc } satisfies DocDragItem, disabled: !menu.moveTo });
  const prefetch = useHoverPrefetch((client) => client.prefetchQuery(docQuery(doc.id)));
  return (
    <ListPaneRow
      to={wsPath(docPath(projectToken, doc.id))}
      title={doc.title}
      dense
      leading={<DocStateIcon runState={runState} locked={!!doc.locked} questionsClosed={!!doc.clarification_closed} className="ml-5" />}
      meta={
        <span title={`${sortBy === "created_at" ? "Created" : "Edited"} ${formatFullTime(doc[sortBy])}`} className="font-mono text-micro text-muted-foreground tabular-nums">
          {formatShortAgo(doc[sortBy])}
        </span>
      }
      selected={selected}
      menu={menu}
      prefetch={prefetch}
      {...(menu.moveTo && { drag })}
    />
  );
};
