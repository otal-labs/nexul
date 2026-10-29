import { personLabel } from "@nexul/client-core/person";

import { DocOverviewRow } from "@/components/doc/DocOverviewRow";
import { usePerson } from "@/hooks/PeopleHooks";
import { useDocRunState } from "@/hooks/TrailHooks";
import type { DocListItem } from "@/models/Doc";

interface DocRecentRowProps {
  doc: DocListItem;
  projectToken: string;
  folderName: string;
}

// The creator, never the last editor; a narrow pane drops the creator and caps the folder so the first line keeps room.
export const DocRecentRow = ({ doc, projectToken, folderName }: DocRecentRowProps) => {
  const runState = useDocRunState(doc.project_id, doc.id);
  const creator = personLabel(usePerson(doc.created_by ?? ""));
  return (
    <DocOverviewRow
      doc={doc}
      runState={runState}
      projectToken={projectToken}
      detail={
        <span className="flex min-w-0">
          <span className="max-w-40 shrink-0 truncate">{folderName}</span>
          <span className="hidden max-w-56 shrink-0 truncate pl-1 @[40rem]:inline">· created by {creator}</span>
          <span className="min-w-0 truncate pl-1">· {doc.snippet || "Empty doc"}</span>
        </span>
      }
    />
  );
};
