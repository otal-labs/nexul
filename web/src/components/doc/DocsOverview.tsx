import { FileTextIcon, PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { DocsAttentionSection } from "@/components/doc/DocsAttentionSection";
import { DocsOverviewHeader } from "@/components/doc/DocsOverviewHeader";
import { docMatches, overviewLists } from "@/components/doc/docsOverviewLists";
import { DocsRecentSection } from "@/components/doc/DocsRecentSection";
import { EmptyState } from "@/components/EmptyState";
import { useDocRunStates } from "@/hooks/TrailHooks";
import { useNewDoc } from "@/hooks/useNewDoc";
import { useDocArchiveStore } from "@/stores/docArchiveStore";
import type { DocListItem } from "@/models/Doc";
import { projectToken, type Project } from "@/models/Project";

const recentHeading = (query: string, showArchived: boolean) => {
  if (query !== "") return "Matching";
  if (showArchived) return "Archived";
  return "Recently edited";
};

interface DocsOverviewProps {
  docs: DocListItem[];
  project: Project;
  /** The list's search, lower-cased and trimmed; both panes show the same matches. */
  query: string;
}

// The open pane before a doc is picked: what needs someone, then what moved lately, from the set the list beside it shows.
export const DocsOverview = ({ docs, project, query }: DocsOverviewProps) => {
  const showArchived = useDocArchiveStore((s) => s.showArchived);
  const listed = docs.filter((doc) => doc.can_open && doc.archived === showArchived);
  const live = docs.filter((doc) => doc.can_open && !doc.archived);
  const shown = query === "" ? listed : listed.filter(docMatches(query));
  const states = useDocRunStates(project.id, shown.map((doc) => doc.id));
  // A search or the archived view reads as one flat set, so nothing is held back for "Needs attention".
  const digest = query === "" && !showArchived;
  const { attention, recent } = overviewLists(shown, digest ? states : []);
  const empty = live.length === 0 && !showArchived;
  const newDoc = useNewDoc(project);
  const token = projectToken(project);
  return (
    <section aria-label="Docs overview" className="@container flex flex-1 flex-col px-6 pt-3 pb-8">
      {empty && (
        <EmptyState
          icon={FileTextIcon}
          title="No docs yet"
          message="Specs, runbooks and meeting notes live here."
          className="m-auto"
          action={
            <ActionButton permission="docs:write" verb="create docs" projectId={project.id} size="sm" icon={PlusIcon} onClick={() => void newDoc()}>
              New doc
            </ActionButton>
          }
        />
      )}
      {!empty && <DocsOverviewHeader docs={listed} project={project} />}
      {attention.length > 0 && <DocsAttentionSection items={attention} projectId={project.id} projectToken={token} />}
      {!empty && (recent.length > 0 || query !== "") && <DocsRecentSection docs={recent} projectId={project.id} projectToken={token} heading={recentHeading(query, showArchived)} />}
    </section>
  );
};
