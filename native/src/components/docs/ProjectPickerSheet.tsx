import { useRouter } from "expo-router";
import { FlatList, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectPickerRow } from "@/components/docs/ProjectPickerRow";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { Project } from "@/models/Project";
import { useDocsProjectStore } from "@/stores/docsProjectStore";

// The form sheet at /more/docs/pick-project. Scoped to Docs for now; Board is building its own project
// picker in parallel, so this stays local instead of a shared component.
export const ProjectPickerSheet = () => {
  const router = useRouter();
  const setSelectedProjectId = useDocsProjectStore((s) => s.setSelectedProjectId);
  const { data: projects, error, isPending } = useFetchProjects();

  const onPick = (project: Project) => {
    setSelectedProjectId(project.id);
    router.back();
  };

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && (
        <FlatList
          data={projects}
          keyExtractor={(project) => project.id}
          renderItem={({ item }) => <ProjectPickerRow project={item} onPress={onPick} />}
        />
      )}
    </View>
  );
};
