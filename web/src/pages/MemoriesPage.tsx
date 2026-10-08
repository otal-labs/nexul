import { lazy, Suspense } from "react";
import { useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MemoriesListPane } from "@/components/memory/MemoriesListPane";
import { NoProjectsState } from "@/components/project/NoProjectsState";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";

// Lazy: the editor's tiptap stack loads only once a memory is open.
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));

// The sidebar's project's memories beside the open one.
export const MemoriesPage = () => {
  const { memoryId } = useParams();
  const { projects, current } = useSidebarProject();
  const { data: memories, error, isPending } = useFetchMemoriesByProject(current?.id ?? "");

  return (
    <div className="h-full">
      {projects && projects.length === 0 && (
        <div className="p-6">
          <NoProjectsState message="Every memory belongs to a project. Create one to start." />
        </div>
      )}
      {current && isPending && <LoadingDisplay label="Loading memories…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load memories." />}
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
