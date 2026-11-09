import { useState, type ReactNode } from "react";

import { DocsListPane } from "@/components/doc/DocsListPane";
import { DocsOverview } from "@/components/doc/DocsOverview";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import type { DocListItem } from "@/models/Doc";
import type { Project } from "@/models/Project";

interface DocsBrowserProps {
  docs: DocListItem[];
  project: Project;
  selectedId: string | undefined;
  /** Built by the page, so a keystroke here hands the open editor the same element and it skips its render. */
  detail: ReactNode;
}

// The doc list and the overview beside it share one search, held here so typing renders these two panes and never the open doc.
export const DocsBrowser = ({ docs, project, selectedId, detail }: DocsBrowserProps) => {
  const [search, setSearch] = useState("");

  return (
    <ListDetailLayout
      hasSelection={!!selectedId}
      list={<DocsListPane docs={docs} project={project} selectedId={selectedId} search={search} onSearch={setSearch} />}
      placeholder={<DocsOverview docs={docs} project={project} query={search.trim().toLowerCase()} />}
      detail={detail}
    />
  );
};
