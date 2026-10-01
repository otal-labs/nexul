import { DocFolderHeader } from "@/components/doc/DocFolderHeader";
import { DocListRow } from "@/components/doc/DocListRow";
import { EmptyRow } from "@/components/EmptyRow";
import type { DocFolderGroup } from "@/components/doc/docGroups";
import { useDocFolderStore } from "@/stores/docFolderStore";

interface DocFolderSectionProps {
  group: DocFolderGroup;
  projectToken: string;
  selectedId: string | undefined;
  /** A running search shows every matching folder open, whatever was collapsed. */
  forceOpen: boolean;
}

export const DocFolderSection = ({ group, projectToken, selectedId, forceOpen }: DocFolderSectionProps) => {
  const { folder } = group;
  const collapsed = useDocFolderStore((s) => (s.collapsed[folder.project_id] ?? []).includes(folder.id));
  const toggle = useDocFolderStore((s) => s.toggleCollapsed);
  const open = forceOpen || !collapsed;
  return (
    <section aria-label={folder.name}>
      <DocFolderHeader folder={folder} total={group.total} open={open} onToggle={() => toggle(folder.project_id, folder.id)} />
      {open && group.docs.length > 0 && (
        <ul className="divide-y divide-border border-b border-border">
          {group.docs.map((doc) => (
            <DocListRow key={doc.id} doc={doc} projectToken={projectToken} selected={doc.id === selectedId} />
          ))}
        </ul>
      )}
      {open && group.docs.length === 0 && (
        <EmptyRow className="rounded-none border-0 border-b py-2 pl-13 text-left text-xs">No docs</EmptyRow>
      )}
    </section>
  );
};
