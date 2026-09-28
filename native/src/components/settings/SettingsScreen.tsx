import { useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { ProfileSection } from "@/components/settings/ProfileSection";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { SignOutButton } from "@/components/settings/SignOutButton";
import { useEnsureWorkspaceSelected, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { appearanceLabel, useAppearanceStore } from "@/stores/appearanceStore";

export const SettingsScreen = () => {
  const router = useRouter();
  const appearance = useAppearanceStore((s) => s.appearance);
  useEnsureWorkspaceSelected();
  const workspace = useSelectedWorkspace();

  return (
    <ScrollView className="flex-1 bg-background">
      <ProfileSection />
      <View className="mt-2">
        <SettingsRow label="Appearance" meta={appearanceLabel[appearance]} onPress={() => router.push("/more/settings/appearance")} />
        <SettingsRow label="Devices" onPress={() => router.push("/more/settings/devices")} />
        <SettingsRow label="Workspace" meta={workspace?.name ?? ""} onPress={() => router.push("/more/settings/workspace")} />
      </View>
      <View className="px-4 py-6">
        <SignOutButton />
      </View>
    </ScrollView>
  );
};
