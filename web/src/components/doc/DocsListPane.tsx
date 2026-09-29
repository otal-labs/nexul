import { useState } from "react";
import { FileTextIcon } from "lucide-react";

import { DocGroupSection } from "@/components/doc/DocGroupSection";
import { groupDocsByDay } from "@/components/doc/docGroups";
import { ListPaneEmpty, ListPaneNoMatch } from "@/components/listpane/ListPaneEmpty";
import { ListPaneHeader } from "@/components/listpane/ListPaneHeader";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import type { DocListItem } from "@/models/Doc";
import { projectToken, type Project } from "@/models/Project";

interface DocsListPaneProps {
  docs: DocListItem[];
  project: Project;
  selectedId: string | undefined;
}

const matches = (doc: DocListItem, query: string) =>
  doc.title.toLowerCase().includes(query) || (doc.snippet ?? "").toLowerCase().includes(query);

// Lists only the docs the viewer can open; the server still names the others, with can_open false.
export const DocsListPane = ({ docs, project, selectedId }: DocsListPaneProps) => {
  const [search, setSearch] = useState("");
  const openCreateDoc = useCreateDocDialog(project.id);
  const openable = docs.filter((doc) => doc.can_open);
  const query = search.trim().toLowerCase();
  const groups = groupDocsByDay(query === "" ? openable : openable.filter((doc) => matches(doc, query)));

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ListPaneHeader
        title="Docs"
        count={openable.length}
        newLabel="New doc"
        onNew={openCreateDoc}
        search={search}
        onSearch={setSearch}
      />
      <div className="min-h-0 flex-1 overflow-y-auto pb-2">
        {openable.length === 0 && <ListPaneEmpty icon={FileTextIcon} message="No docs yet" onNew={openCreateDoc} />}
        {openable.length > 0 && groups.length === 0 && <ListPaneNoMatch onClear={() => setSearch("")} />}
        {groups.map((group) => (
          <DocGroupSection key={group.label} group={group} projectToken={projectToken(project)} selectedId={selectedId} />
        ))}
      </div>
    </div>
  );
};
