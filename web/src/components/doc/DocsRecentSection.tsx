import { DocDayGroup } from "@/components/doc/DocDayGroup";
import { dayGroups } from "@/components/doc/docsOverviewLists";
import { DocRecentRow } from "@/components/doc/DocRecentRow";
import { microheaderClass } from "@/components/Microheader";
import { useDocFolderNames } from "@/hooks/DocFolderHooks";
import type { DocListItem } from "@/models/Doc";
import { cn } from "@/lib/utils";

interface DocsRecentSectionProps {
  docs: DocListItem[];
  projectId: string;
  projectToken: string;
  /** "Recently edited", "Archived", or "Matching" while the list's search runs. */
  heading: string;
}

// Recent edits are headed by their day, so someone coming back stops at what they have seen; a search or the archived view is one flat set under its own head.
export const DocsRecentSection = ({ docs, projectId, projectToken, heading }: DocsRecentSectionProps) => {
  const folderName = useDocFolderNames(projectId);
  const byDay = heading === "Recently edited";
  return (
    <section aria-label={heading}>
      {byDay && dayGroups(docs).map((group) => <DocDayGroup key={group.label} label={group.label} docs={group.docs} projectToken={projectToken} folderName={folderName} />)}
      {!byDay && (
        <>
          <h2 className={cn(microheaderClass, "flex items-baseline gap-2 pb-1")}>
            {heading}
            {heading === "Matching" && <span className="tabular-nums">{docs.length}</span>}
          </h2>
          <ul>
            {docs.map((doc) => (
              <DocRecentRow key={doc.id} doc={doc} projectToken={projectToken} folderName={folderName(doc.folder_id)} />
            ))}
          </ul>
        </>
      )}
    </section>
  );
};
