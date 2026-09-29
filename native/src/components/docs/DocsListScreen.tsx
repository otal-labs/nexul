import { useRouter } from "expo-router";
import { View } from "react-native";

import { DocsFeed } from "@/components/docs/DocsFeed";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { DocListItem } from "@/models/Doc";
import { effectiveProject } from "@/models/Project";
import { useDocsProjectStore } from "@/stores/docsProjectStore";

export const DocsListScreen = () => {
  const router = useRouter();
  const selectedProjectId = useDocsProjectStore((s) => s.selectedProjectId);
  const { data: projects, error: projectsError, isPending: projectsPending } = useFetchProjects();
  const activeProject = effectiveProject(projects, selectedProjectId);
  const activeProjectId = activeProject?.id;
  const {
    data: docs,
    error: docsError,
    isPending: docsPending,
    isRefetching,
    refetch,
  } = useFetchDocsByProject(activeProjectId);

  const onSelect = (doc: DocListItem) => {
    if (doc.can_open) router.push(`/more/docs/${doc.id}`);
  };

  return (
    <View className="flex-1 bg-background">
      <View className="flex-row items-center justify-between border-b border-border px-4 py-2">
        <Text className="font-medium">{activeProject?.name ?? "Choose a project"}</Text>
        {projects && projects.length > 1 && (
          <Button variant="ghost" size="sm" onPress={() => router.push("/more/docs/pick-project")}>
            <Text>Change</Text>
          </Button>
        )}
      </View>
      {projectsPending && <LoadingDisplay />}
      {projectsError && <ErrorDisplay error={projectsError} />}
      {projects && projects.length === 0 && <PlaceholderScreen message="No projects yet." />}
      {activeProjectId && docsPending && <LoadingDisplay />}
      {activeProjectId && docsError && <ErrorDisplay error={docsError} />}
      {activeProjectId && docs && docs.length === 0 && <PlaceholderScreen message="No docs yet." />}
      {activeProjectId && docs && docs.length > 0 && (
        <DocsFeed docs={docs} refreshing={isRefetching} onRefresh={() => void refetch()} onSelect={onSelect} />
      )}
    </View>
  );
};
