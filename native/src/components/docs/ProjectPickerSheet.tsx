import { useRouter } from "expo-router";
import { FlatList, View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectPickerRow } from "@/components/ProjectPickerRow";
import { SheetTitle } from "@/components/SheetTitle";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { effectiveProject, type Project } from "@/models/Project";
import { useDocsProjectStore } from "@/stores/docsProjectStore";

// The form sheet at /more/docs/pick-project.
export const ProjectPickerSheet = () => {
  const router = useRouter();
  const selectedProjectId = useDocsProjectStore((s) => s.selectedProjectId);
  const setSelectedProjectId = useDocsProjectStore((s) => s.setSelectedProjectId);
  const { data: projects, error, isPending } = useFetchProjects();

  const currentId = effectiveProject(projects, selectedProjectId)?.id;

  const onPick = (project: Project) => {
    setSelectedProjectId(project.id);
    router.back();
  };

  return (
    <View className="bg-popover pb-6">
      <SheetTitle title="Docs project" />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && (
        <FlatList
          data={projects}
          keyExtractor={(project) => project.id}
          renderItem={({ item }) => <ProjectPickerRow project={item} selected={item.id === currentId} onPress={onPick} />}
        />
      )}
    </View>
  );
};
