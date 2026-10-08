import { useEffect, useState } from "react";
import { FileTextIcon } from "lucide-react";

import { DocFolderSection } from "@/components/doc/DocFolderSection";
import { DocGroupSection } from "@/components/doc/DocGroupSection";
import { DocSortToggle } from "@/components/doc/DocSortToggle";
import { groupDocs } from "@/components/doc/docGroups";
import { NewDocFolderButton } from "@/components/doc/NewDocFolderButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneEmpty, ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import { useDocPins } from "@/hooks/useDocPins";
import { useDocFolderStore } from "@/stores/docFolderStore";
import { useDocSortStore } from "@/stores/docSortStore";
import type { DocListItem } from "@/models/Doc";
import { projectToken, type Project } from "@/models/Project";

interface DocsListPaneProps {
  docs: DocListItem[];
  project: Project;
  selectedId: string | undefined;
}

const matches = (query: string) => (doc: DocListItem) =>
  doc.title.toLowerCase().includes(query) || (doc.snippet ?? "").toLowerCase().includes(query);

// Lists only the docs the viewer can open, by folder; the server still names the others, with can_open false.
export const DocsListPane = ({ docs, project, selectedId }: DocsListPaneProps) => {
  const [search, setSearch] = useState("");
  const sortBy = useDocSortStore((s) => s.sortBy);
  const { pinnedIds } = useDocPins();
  const openCreateDoc = useCreateDocDialog(project.id);
  const { data: folders, error, isPending } = useFetchDocFolders(project.id);
  const openable = docs.filter((doc) => doc.can_open);
  const query = search.trim().toLowerCase();
  const searching = query !== "";
  const groups = folders && groupDocs({ docs: openable, folders, sortBy, pinnedIds, match: searching ? matches(query) : undefined });
  const token = projectToken(project);
  const expandFolder = useDocFolderStore((s) => s.expand);
  const selectedFolderId = docs.find((doc) => doc.id === selectedId)?.folder_id;

  // Opening a doc shows where it lives; collapsing that folder afterwards still sticks.
  useEffect(() => {
    if (selectedFolderId) expandFolder(project.id, selectedFolderId);
  }, [selectedId, selectedFolderId, project.id, expandFolder]);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Docs"
        count={openable.length}
        newLabel="New doc"
        onNew={openCreateDoc}
        search={search}
        onSearch={setSearch}
        controls={
          <>
            <DocSortToggle />
            <NewDocFolderButton projectId={project.id} />
          </>
        }
      />
      <div className="min-h-0 flex-1 overflow-y-auto pb-2">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} title="Couldn't load folders." />}
        {groups && groups.folders.length === 0 && !searching && <ListPaneEmpty icon={FileTextIcon} message="No docs yet" />}
        {groups && groups.folders.length === 0 && groups.pinned.length === 0 && searching && <ListPaneNoMatch onClear={() => setSearch("")} />}
        {groups && groups.pinned.length > 0 && (
          <DocGroupSection group={{ label: "Pinned", docs: groups.pinned }} projectToken={token} selectedId={selectedId} />
        )}
        {groups?.folders.map((group) => (
          <DocFolderSection key={group.folder.id} group={group} projectToken={token} selectedId={selectedId} forceOpen={searching} />
        ))}
      </div>
    </div>
  );
};
