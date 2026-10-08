import { Link } from "react-router";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Doc } from "@/models/Doc";

interface DocBreadcrumbProps {
  doc: Doc;
}

export const DocBreadcrumb = ({ doc }: DocBreadcrumbProps) => {
  const wsPath = useWorkspacePath();
  const { data: folders } = useFetchDocFolders(doc.project_id);
  const folder = folders?.find((f) => f.id === doc.folder_id);
  return (
    <Breadcrumb>
      <BreadcrumbList className="flex-nowrap gap-1.5 font-mono text-xs sm:gap-1.5">
        <BreadcrumbItem className="shrink-0">
          <BreadcrumbLink asChild className="duration-150 ease-standard">
            <Link to={wsPath("/docs")}>Docs</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        {folder && (
          <>
            <BreadcrumbSeparator />
            <BreadcrumbItem className="min-w-0 max-w-1/2 shrink-0">
              <span className="truncate">{folder.name}</span>
            </BreadcrumbItem>
          </>
        )}
        <BreadcrumbSeparator />
        <BreadcrumbItem className="min-w-0">
          <BreadcrumbPage className="truncate">{doc.title}</BreadcrumbPage>
        </BreadcrumbItem>
      </BreadcrumbList>
    </Breadcrumb>
  );
};
