import { DocAttentionRow } from "@/components/doc/DocAttentionRow";
import type { AttentionDoc } from "@/components/doc/docsOverviewLists";
import { microheaderClass } from "@/components/Microheader";
import { useDocFolderNames } from "@/hooks/DocFolderHooks";
import { cn } from "@/lib/utils";

interface DocsAttentionSectionProps {
  items: AttentionDoc[];
  projectId: string;
  projectToken: string;
}

// The docs a play is writing or waiting on, so they never hide in a collapsed folder.
export const DocsAttentionSection = ({ items, projectId, projectToken }: DocsAttentionSectionProps) => {
  const folderName = useDocFolderNames(projectId);
  return (
    <section aria-label="Needs attention" className="pb-6">
      <h2 className={cn(microheaderClass, "pb-1")}>Needs attention</h2>
      <ul>
        {items.map((item) => (
          <DocAttentionRow key={item.doc.id} item={item} projectToken={projectToken} folderName={folderName(item.doc.folder_id)} />
        ))}
      </ul>
    </section>
  );
};
