import { DocRecentRow } from "@/components/doc/DocRecentRow";
import { microheaderClass } from "@/components/Microheader";
import type { DocListItem } from "@/models/Doc";
import { cn } from "@/lib/utils";

interface DocDayGroupProps {
  label: string;
  docs: DocListItem[];
  projectToken: string;
  folderName: (folderId: string) => string;
}

export const DocDayGroup = ({ label, docs, projectToken, folderName }: DocDayGroupProps) => (
  <section aria-label={label} className="pb-6">
    <h2 className={cn(microheaderClass, "pb-1")}>{label}</h2>
    <ul>
      {docs.map((doc) => (
        <DocRecentRow key={doc.id} doc={doc} projectToken={projectToken} folderName={folderName(doc.folder_id)} />
      ))}
    </ul>
  </section>
);
