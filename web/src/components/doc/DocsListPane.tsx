import { useEffect } from "react";
import { ArchiveIcon, FileTextIcon, PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { DocFolderSection } from "@/components/doc/DocFolderSection";
import { DocGroupSection } from "@/components/doc/DocGroupSection";
import { groupDocs } from "@/components/doc/docGroups";
import { DocListMenu } from "@/components/doc/DocListMenu";
import { docMatches, overviewLists } from "@/components/doc/docsOverviewLists";
import { NewDocFolderButton } from "@/components/doc/NewDocFolderButton";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneEmpty, ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useDocRunStates } from "@/hooks/TrailHooks";
import { useDocPins } from "@/hooks/useDocPins";
import { useNewDoc } from "@/hooks/useNewDoc";
import { useDocArchiveStore } from "@/stores/docArchiveStore";
import { useDocFolderStore } from "@/stores/docFolderStore";
import { useDocSortStore } from "@/stores/docSortStore";
import type { DocListItem } from "@/models/Doc";
import { projectToken, type Project } from "@/models/Project";

interface DocsListPaneProps {
  docs: DocListItem[];
  project: Project;
  selectedId: string | undefined;
  /** The page's search, shared with the overview beside the list. */
  search: string;
  onSearch: (value: string) => void;
}

// A search lists the folders with a match; the archived view lists the folders holding something, so empty ones don't pile up.
const folderFilter = (query: string, showArchived: boolean) => {
  if (query !== "") return docMatches(query);
  if (showArchived) return () => true;
  return undefined;
};

// Lists only the docs the viewer can open, by folder; the server still names the others, with can_open false.
export const DocsListPane = ({ docs, project, selectedId, search, onSearch }: DocsListPaneProps) => {
  const sortBy = useDocSortStore((s) => s.sortBy);
  const showArchived = useDocArchiveStore((s) => s.showArchived);
  const { pinnedIds } = useDocPins();
  const newDoc = useNewDoc(project);
  const { data: folders, error, isPending } = useFetchDocFolders(project.id);
  const openable = docs.filter((doc) => doc.can_open && doc.archived === showArchived);
  const query = search.trim().toLowerCase();
  const searching = query !== "";
  const groups = folders && groupDocs({ docs: openable, folders, sortBy, pinnedIds, match: folderFilter(query, showArchived) });
  const token = projectToken(project);
  // Below 1024 the list is the whole page, so the docs a play holds lead it, wherever their folder is.
  const states = useDocRunStates(project.id, openable.map((doc) => doc.id));
  const attention = overviewLists(openable, states).attention.map((item) => item.doc);
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
        newPermission="docs:write"
        newVerb="create docs"
        projectId={project.id}
        onNew={() => void newDoc()}
        search={search}
        {...(docs.some((doc) => doc.can_open) && { onSearch })}
        controls={
          <>
            <DocListMenu />
            <NewDocFolderButton projectId={project.id} />
          </>
        }
      />
      <div className="scroll-edge min-h-0 flex-1 overflow-y-auto pb-6">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} title="Couldn't load folders." />}
        {groups && openable.length === 0 && !searching && !showArchived && (
          // From 1024 the overview beside the list says this, with its action.
          <div className="flex h-full items-center justify-center lg:hidden">
            <EmptyState
              icon={FileTextIcon}
              title="No docs yet"
              className="border-0"
              message="Specs, runbooks and meeting notes live here."
              action={
                <ActionButton permission="docs:write" verb="create docs" projectId={project.id} size="sm" icon={PlusIcon} onClick={() => void newDoc()}>
                  New doc
                </ActionButton>
              }
            />
          </div>
        )}
        {groups && groups.folders.length === 0 && groups.pinned.length === 0 && !searching && showArchived && <ListPaneEmpty icon={ArchiveIcon} message="No archived docs" />}
        {groups && groups.folders.length === 0 && groups.pinned.length === 0 && searching && <ListPaneNoMatch onClear={() => onSearch("")} />}
        {groups && attention.length > 0 && !searching && (
          <div className="lg:hidden">
            <DocGroupSection group={{ label: "Needs attention", docs: attention }} projectToken={token} selectedId={selectedId} />
          </div>
        )}
        {groups && groups.pinned.length > 0 && (
          <div className="pb-2">
            <DocGroupSection group={{ label: "Pinned", docs: groups.pinned }} projectToken={token} selectedId={selectedId} />
          </div>
        )}
        {groups && (openable.length > 0 || searching || showArchived) && groups.folders.map((group) => (
          <DocFolderSection key={group.folder.id} group={group} projectToken={token} selectedId={selectedId} forceOpen={searching} />
        ))}
      </div>
    </div>
  );
};
