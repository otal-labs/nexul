import { useRouter } from "expo-router";
import { FlatList } from "react-native";

import { SheetBody } from "@/components/SheetBody";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectPickerRow } from "@/components/ProjectPickerRow";
import { SheetTitle } from "@/components/SheetTitle";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { effectiveProject } from "@/models/Project";
import { useBoardStore } from "@/stores/boardStore";

export const ProjectPickerScreen = () => {
  const router = useRouter();
  const { data: projects, error, isPending } = useFetchProjects();
  const selectedProjectId = useBoardStore((s) => s.selectedProjectId);
  const selectProject = useBoardStore((s) => s.selectProject);
  const currentId = effectiveProject(projects, selectedProjectId)?.id;

  return (
    <SheetBody>
      <SheetTitle title="Board project" />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && (
        <FlatList
          data={projects}
          keyExtractor={(p) => p.id}
          renderItem={({ item }) => (
            <ProjectPickerRow
              project={item}
              selected={item.id === currentId}
              onPress={(project) => {
                selectProject(project.id);
                router.back();
              }}
            />
          )}
        />
      )}
    </SheetBody>
  );
};
