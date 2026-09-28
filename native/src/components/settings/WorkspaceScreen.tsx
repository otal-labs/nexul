import { useRouter } from "expo-router";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { WorkspaceRow } from "@/components/settings/WorkspaceRow";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const WorkspaceScreen = () => {
  const router = useRouter();
  const { data, error, isPending } = useFetchWorkspaces();
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectWorkspace = useWorkspaceStore((s) => s.selectWorkspace);

  const choose = (id: string) => {
    selectWorkspace(id);
    router.back();
  };

  return (
    <View className="flex-1 bg-background pt-2">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <PlaceholderScreen message="No workspaces yet." />}
      {data &&
        data.length > 0 &&
        data.map((workspace) => (
          <WorkspaceRow
            key={workspace.id}
            workspace={workspace}
            selected={workspace.id === selectedWorkspaceId}
            onPress={() => choose(workspace.id)}
          />
        ))}
    </View>
  );
};
