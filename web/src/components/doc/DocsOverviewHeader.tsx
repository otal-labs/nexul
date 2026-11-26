import { PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { formatShortAgo } from "@/components/doc/docTime";
import { PageHeader } from "@/components/PageHeader";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useNewDoc } from "@/hooks/useNewDoc";
import type { DocListItem } from "@/models/Doc";
import type { Project } from "@/models/Project";

interface DocsOverviewHeaderProps {
  docs: DocListItem[];
  project: Project;
}

// The pane's title over what the list beside it does not say (how many folders, how fresh), and its one primary action.
export const DocsOverviewHeader = ({ docs, project }: DocsOverviewHeaderProps) => {
  const { data: folders } = useFetchDocFolders(project.id);
  const newDoc = useNewDoc(project);
  const lastEdit = docs.reduce((latest, doc) => (doc.updated_at > latest ? doc.updated_at : latest), "");
  const folderCount = folders?.length ?? 0;
  const ago = formatShortAgo(lastEdit);
  // Only a relative figure is mono, so "14m ago" keeps a word space and "8 Oct" reads as words.
  const relative = /\d[mhd]$/.test(ago);
  return (
    <PageHeader
      title="Docs"
      className="mb-5"
      meta={
        <p className="truncate">
          {folders && (
            <>
              <span className="font-mono tabular-nums">{folderCount}</span> {folderCount === 1 ? "folder" : "folders"}
            </>
          )}
          {folders && lastEdit !== "" && " · "}
          {lastEdit !== "" && (
            <>
              last edit{" "}
              {relative && (
                <>
                  <span className="font-mono tabular-nums">{ago}</span> ago
                </>
              )}
              {!relative && ago}
            </>
          )}
        </p>
      }
      actions={
        <ActionButton permission="docs:write" verb="create docs" projectId={project.id} size="sm" icon={PlusIcon} onClick={() => void newDoc()}>
          New doc
        </ActionButton>
      }
    />
  );
};
