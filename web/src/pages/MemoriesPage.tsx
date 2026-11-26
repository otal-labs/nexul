import { lazy, Suspense } from "react";
import { Navigate, useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoriesListPane } from "@/components/memory/MemoriesListPane";
import { MemoriesPlaceholder } from "@/components/memory/MemoriesPlaceholder";
import { groupMemoriesByFolder } from "@/components/memory/memoryGroups";
import { NoProjectsState } from "@/components/project/NoProjectsState";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useMinWidth } from "@/hooks/useMinWidth";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { memoryPath, projectToken } from "@/models/Project";

// Lazy: the editor's tiptap stack loads only once a memory is open.
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));

// The sidebar's project's memories beside the open one.
export const MemoriesPage = () => {
  const { memoryId } = useParams();
  const { projects, current } = useSidebarProject();
  const { data: memories, error, isPending } = useFetchMemoriesByProject(current?.id ?? "");
  // From 1024px the open pane sits beside the list, so it opens on the list's first memory rather than an empty pane.
  const wide = useMinWidth(1024);
  const wsPath = useWorkspacePath();
  const first = wide && !memoryId && memories ? groupMemoriesByFolder(memories).find((g) => g.memories.length > 0)?.memories[0] : undefined;

  return (
    <div className="h-full">
      {projects && projects.length === 0 && (
        <div className="p-6">
          <NoProjectsState message="Every memory belongs to a project. Create one to start." />
        </div>
      )}
      {current && first && <Navigate replace to={wsPath(memoryPath(projectToken(current), first.id))} />}
      {current && isPending && <LoadingDisplay label="Loading memories…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load memories." />}
      {current && memories && (
        <ListDetailLayout
          hasSelection={!!memoryId}
          stableGutter
          list={<MemoriesListPane memories={memories} project={current} selectedId={memoryId} />}
          placeholder={<MemoriesPlaceholder memories={memories} project={current} />}
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
