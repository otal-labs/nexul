import { FileTextIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import { Navigate, useParams } from "react-router";

import { DocsListPane } from "@/components/doc/DocsListPane";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoProjectsState } from "@/components/project/NoProjectsState";
import { useFetchDoc, useFetchDocsByProject } from "@/hooks/DocHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { docPath, projectTokenById } from "@/models/Project";

// Lazy: the editor's tiptap and yjs stack loads only once a doc is open.
const DocPage = lazy(() => import("@/pages/DocPage").then((m) => ({ default: m.DocPage })));

// The sidebar's project's docs beside the open one; a bare /docs/:docId link moves to its project's URL.
export const DocsPage = () => {
  const { projectToken, docId } = useParams();
  const { projects, current } = useSidebarProject();
  const wsPath = useWorkspacePath();
  const { data: docs, error, isPending } = useFetchDocsByProject(current?.id ?? "");
  const { data: bareDoc } = useFetchDoc(projectToken ? undefined : docId);

  return (
    <div className="h-full">
      {bareDoc && projects && <Navigate replace to={wsPath(docPath(projectTokenById(projects, bareDoc.project_id), bareDoc.id))} />}
      {projects && projects.length === 0 && (
        <div className="p-6">
          <NoProjectsState message="Every doc belongs to a project. Create one to start writing." />
        </div>
      )}
      {current && isPending && <LoadingDisplay label="Loading docs…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load docs." />}
      {current && docs && (
        <ListDetailLayout
          hasSelection={!!docId}
          list={<DocsListPane docs={docs} project={current} selectedId={docId} />}
          placeholder={<ListDetailPlaceholder icon={FileTextIcon} title="Select a doc" summary={`${docs.length} ${docs.length === 1 ? "doc" : "docs"} in ${current.name}`} searchable />}
          detail={
            <Suspense fallback={<LoadingDisplay />}>
              <DocPage key={docId} />
            </Suspense>
          }
        />
      )}
    </div>
  );
};
