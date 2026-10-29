import { DocRecentRow } from "@/components/doc/DocRecentRow";
import type { DocListItem } from "@/models/Doc";

interface DocDayGroupProps {
  /** Left out when every row falls on one side of the day lines. */
  label?: string;
  docs: DocListItem[];
  projectToken: string;
  folderName: (folderId: string) => string;
}

export const DocDayGroup = ({ label, docs, projectToken, folderName }: DocDayGroupProps) => (
  <section aria-label={label} className="pb-3">
    {label && <h3 className="pt-2 pb-1 text-xs font-medium text-muted-foreground">{label}</h3>}
    <ul>
      {docs.map((doc) => (
        <DocRecentRow key={doc.id} doc={doc} projectToken={projectToken} folderName={folderName(doc.folder_id)} />
      ))}
    </ul>
  </section>
);
