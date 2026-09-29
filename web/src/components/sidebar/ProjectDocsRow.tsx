import { FileTextIcon } from "lucide-react";

import { SidebarNavLink } from "@/components/sidebar/SidebarNavLink";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { docPath, projectToken, type Project } from "@/models/Project";

interface ProjectDocsRowProps {
  project: Project;
  collapsed: boolean;
}

// No "Docs" label/toggle row — each doc title is a direct sibling of Board, distinct icon per DocRow.tsx.
export const ProjectDocsRow = ({ project, collapsed }: ProjectDocsRowProps) => {
  const { data: docs = [] } = useFetchDocsByProject(project.id);
  // The list names docs the viewer can't open (Docs shows them locked); the sidebar only links what opens.
  const openable = docs.filter((doc) => doc.can_open);

  return (
    <>
      {openable.map((doc) => (
        <SidebarNavLink
          key={doc.id}
          to={docPath(projectToken(project), doc.id)}
          label={doc.title}
          icon={FileTextIcon}
          collapsed={collapsed}
        />
      ))}
    </>
  );
};
