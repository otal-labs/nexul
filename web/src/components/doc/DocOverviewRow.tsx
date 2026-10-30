import type { ReactNode } from "react";
import { Link } from "react-router";

import { DocStateIcon } from "@/components/doc/DocStateIcon";
import { formatShortAgo } from "@/components/doc/docTime";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { DocListItem } from "@/models/Doc";
import { docPath } from "@/models/Project";
import type { TrailState } from "@/models/Trail";
import { formatFullTime } from "@/utils/TimeUtility";

interface DocOverviewRowProps {
  doc: DocListItem;
  runState: TrailState | undefined;
  projectToken: string;
  /** The muted second line: where the doc lives and what it holds or waits on. */
  detail: ReactNode;
}

// The overview's one row shape: the state slot, the title on the time's baseline, a muted detail line under it.
export const DocOverviewRow = ({ doc, runState, projectToken, detail }: DocOverviewRowProps) => {
  const wsPath = useWorkspacePath();
  return (
    <li>
      <Link
        to={wsPath(docPath(projectToken, doc.id))}
        className="-mx-2 flex gap-2 rounded-md px-2 py-2 text-[13px] transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
      >
        <span className="flex h-lh shrink-0 items-center">
          <DocStateIcon runState={runState} locked={!!doc.locked} questionsClosed={!!doc.clarification_closed} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex items-baseline gap-1.5">
            <span dir="auto" title={doc.title} className="truncate font-medium text-foreground">
              {doc.title}
            </span>
            <span title={`Edited ${formatFullTime(doc.updated_at)}`} className="ml-auto shrink-0 pl-3 font-mono text-[11px] text-muted-foreground tabular-nums">
              {formatShortAgo(doc.updated_at)}
            </span>
          </span>
          <span dir="auto" className="block truncate text-xs text-muted-foreground">
            {detail}
          </span>
        </span>
      </Link>
    </li>
  );
};
