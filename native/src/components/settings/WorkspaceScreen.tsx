import { useNavigation, useRouter } from "expo-router";
import type { NavigationProp, ParamListBase } from "expo-router/react-navigation";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { SheetTitle } from "@/components/SheetTitle";
import { WorkspaceRow } from "@/components/settings/WorkspaceRow";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const WorkspaceScreen = () => {
  const router = useRouter();
  const navigation = useNavigation<NavigationProp<ParamListBase>>();
  const { data, error, isPending } = useFetchWorkspaces();
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectWorkspace = useWorkspaceStore((s) => s.selectWorkspace);

  // Resetting the root stack drops the sheet and remounts every tab at its root, so no old-workspace screen survives.
  const choose = (id: string) => {
    if (id === selectedWorkspaceId) {
      router.back();
      return;
    }
    selectWorkspace(id);
    navigation.reset({ index: 0, routes: [{ name: "(tabs)", state: { routes: [{ name: "more" }] } }] });
  };

  return (
    <View role="radiogroup" className="bg-popover pb-6">
      <SheetTitle title="Switch workspace" />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <View className="px-5"><EmptyRow message="No workspaces yet." /></View>}
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
