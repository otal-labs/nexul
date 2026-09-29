import { useNavigation, useRouter } from "expo-router";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { SheetTitle } from "@/components/SheetTitle";
import { WorkspaceRow } from "@/components/settings/WorkspaceRow";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const WorkspaceScreen = () => {
  const router = useRouter();
  const tabs = useNavigation("/(tabs)");
  const { data, error, isPending } = useFetchWorkspaces();
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectWorkspace = useWorkspaceStore((s) => s.selectWorkspace);

  // A fresh tab navigator starts every tab at its root, so no old-workspace ticket, thread, or log survives the switch.
  const choose = (id: string) => {
    if (id === selectedWorkspaceId) {
      router.back();
      return;
    }
    selectWorkspace(id);
    tabs.getParent()?.reset({ index: 0, routes: [{ name: "(tabs)", state: { routes: [{ name: "more" }] } }] });
  };

  return (
    <View className="bg-popover pb-6">
      <SheetTitle title="Switch workspace" />
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
