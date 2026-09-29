import { lazy, Suspense } from "react";
import { Navigate, useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoriesListPane } from "@/components/memory/MemoriesListPane";
import { NoProjectsState } from "@/components/project/NoProjectsState";
import { useFetchMemoriesByProject, useFetchMemory } from "@/hooks/MemoryHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { memoryPath, projectTokenById } from "@/models/Project";

// Lazy: the editor's tiptap stack loads only once a memory is open.
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));

// The sidebar's project's memories beside the open one. /memories/:memoryId is a workspace memory's own URL; a
// project memory reached that way moves to its project's.
export const MemoriesPage = () => {
  const { projectToken, memoryId } = useParams();
  const { projects, current } = useSidebarProject();
  const { data: memories, error, isPending } = useFetchMemoriesByProject(current?.id ?? "");
  const { data: bareMemory } = useFetchMemory(projectToken ? undefined : memoryId);
  const movedProjectId = bareMemory?.project_id ?? "";

  return (
    <div>
      {bareMemory && movedProjectId !== "" && projects && (
        <Navigate replace to={memoryPath(projectTokenById(projects, movedProjectId), bareMemory.id)} />
      )}
      {projects && projects.length === 0 && (
        <div className="p-6">
          <NoProjectsState message="Memories live in a project or its workspace. Create a project to start." />
        </div>
      )}
      {current && isPending && <LoadingDisplay label="Loading memories…" />}
      {error && <ErrorDisplay error={error} title="Failed to load memories." />}
      {current && memories && (
        <ListDetailLayout
          hasSelection={!!memoryId}
          list={<MemoriesListPane memories={memories} project={current} selectedId={memoryId} />}
          placeholder="Select a memory"
          detail={
            <Suspense fallback={<LoadingDisplay />}>
              <MemoryPage key={memoryId} />
            </Suspense>
          }
        />
      )}
    </div>
  );
};
