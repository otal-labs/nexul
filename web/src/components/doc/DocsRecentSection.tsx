import { DocDayGroup } from "@/components/doc/DocDayGroup";
import { dayGroups } from "@/components/doc/docsOverviewLists";
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

// Grouped by day so someone coming back stops at what they have seen; one group needs no label of its own.
export const DocsRecentSection = ({ docs, projectId, projectToken, heading }: DocsRecentSectionProps) => {
  const folderName = useDocFolderNames(projectId);
  const groups = dayGroups(docs);
  return (
    <section aria-label={heading}>
      <h2 className={cn(microheaderClass, "flex items-baseline gap-2 pb-1")}>
        {heading}
        {heading === "Matching" && <span className="tabular-nums">{docs.length}</span>}
      </h2>
      {groups.map((group) => (
        <DocDayGroup key={group.label} {...(groups.length > 1 && { label: group.label })} docs={group.docs} projectToken={projectToken} folderName={folderName} />
      ))}
    </section>
  );
};
