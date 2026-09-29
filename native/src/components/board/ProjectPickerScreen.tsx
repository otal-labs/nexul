import { useRouter } from "expo-router";
import { Check } from "lucide-react-native";
import { FlatList, Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useBoardStore } from "@/stores/boardStore";

export const ProjectPickerScreen = () => {
  const router = useRouter();
  const [foreground] = useCSSVariable(["--color-foreground"]);
  const { data: projects, error, isPending } = useFetchProjects();
  const selectedProjectId = useBoardStore((s) => s.selectedProjectId);
  const selectProject = useBoardStore((s) => s.selectProject);

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && (
        <FlatList
          data={projects}
          keyExtractor={(p) => p.id}
          renderItem={({ item }) => (
            <Pressable
              role="button"
              onPress={() => {
                selectProject(item.id);
                router.back();
              }}
              className="min-h-11 flex-row items-center gap-2 border-b border-border px-4 active:bg-accent"
            >
              <Text className="min-w-0 flex-1 font-medium" numberOfLines={1}>
                {item.name}
              </Text>
              <Text variant="small" className="shrink-0 font-mono text-muted-foreground">
                {item.prefix}
              </Text>
              {item.id === selectedProjectId && <Check size={16} color={String(foreground)} />}
            </Pressable>
          )}
        />
      )}
    </View>
  );
};
