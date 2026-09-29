import { DocListRow } from "@/components/doc/DocListRow";
import type { DocGroup } from "@/components/doc/docGroups";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";

interface DocGroupSectionProps {
  group: DocGroup;
  projectToken: string;
  selectedId: string | undefined;
}

export const DocGroupSection = ({ group, projectToken, selectedId }: DocGroupSectionProps) => (
  <ListPaneGroup label={group.label}>
    {group.docs.map((doc) => (
      <DocListRow key={doc.id} doc={doc} projectToken={projectToken} selected={doc.id === selectedId} />
    ))}
  </ListPaneGroup>
);
