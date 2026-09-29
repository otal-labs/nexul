import { lazy, Suspense } from "react";
import { Navigate, useParams } from "react-router";

import { DocsListPane } from "@/components/doc/DocsListPane";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoProjectsState } from "@/components/project/NoProjectsState";
import { useFetchDoc, useFetchDocsByProject } from "@/hooks/DocHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { docPath, projectTokenById } from "@/models/Project";

// Lazy: the editor's tiptap and yjs stack loads only once a doc is open.
const DocPage = lazy(() => import("@/pages/DocPage").then((m) => ({ default: m.DocPage })));

// The sidebar's project's docs beside the open one; a bare /docs/:docId link moves to its project's URL.
export const DocsPage = () => {
  const { projectToken, docId } = useParams();
  const { projects, current } = useSidebarProject();
  const { data: docs, error, isPending } = useFetchDocsByProject(current?.id ?? "");
  const { data: bareDoc } = useFetchDoc(projectToken ? undefined : docId);

  return (
    <div>
      {bareDoc && projects && <Navigate replace to={docPath(projectTokenById(projects, bareDoc.project_id), bareDoc.id)} />}
      {projects && projects.length === 0 && (
        <div className="p-6">
          <NoProjectsState message="Every doc belongs to a project. Create one to start writing." />
        </div>
      )}
      {current && isPending && <LoadingDisplay label="Loading docs…" />}
      {error && <ErrorDisplay error={error} title="Failed to load docs." />}
      {current && docs && (
        <ListDetailLayout
          hasSelection={!!docId}
          list={<DocsListPane docs={docs} project={current} selectedId={docId} />}
          placeholder="Select a doc"
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
