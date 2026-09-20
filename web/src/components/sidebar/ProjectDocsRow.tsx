import { FileTextIcon } from "lucide-react";
import { NavLink } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { docPath, projectToken, type Project } from "@/models/Project";
import { cn } from "@/lib/utils";

interface ProjectDocsRowProps {
  project: Project;
  /** Icon-only, no title text — for the collapsed rail's expanded flyout. */
  collapsed?: boolean;
}

// No "Docs" label/toggle row — each doc title is a direct sibling of Board, distinct icon per DocRow.tsx.
export const ProjectDocsRow = ({ project, collapsed = false }: ProjectDocsRowProps) => {
  const { data: docs = [] } = useFetchDocsByProject(project.id);

  return (
    <>
      {docs.map((doc) => (
        <NavLink
          key={doc.id}
          to={docPath(projectToken(project), doc.id)}
          title={collapsed ? doc.title : undefined}
          className={({ isActive }) => cn(navLinkClass({ isActive }), collapsed && "justify-center")}
        >
          <FileTextIcon className="size-4 shrink-0" aria-hidden />
          {!collapsed && <span className="truncate">{doc.title}</span>}
        </NavLink>
      ))}
    </>
  );
};
