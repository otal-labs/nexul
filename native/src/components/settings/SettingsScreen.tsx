import { useRouter } from "expo-router";
import Building2 from "lucide-react-native/icons/building-complex";
import MonitorSmartphone from "lucide-react-native/icons/monitor-smartphone";
import SunMoon from "lucide-react-native/icons/sun-moon";
import { ScrollView, View } from "react-native";

import { ScreenHeader } from "@/components/ScreenHeader";
import { SettingsCard } from "@/components/SettingsCard";
import { ProfileSection } from "@/components/settings/ProfileSection";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { SignOutButton } from "@/components/settings/SignOutButton";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { appearanceLabel, useAppearanceStore } from "@/stores/appearanceStore";

export const SettingsScreen = () => {
  const router = useRouter();
  const appearance = useAppearanceStore((s) => s.appearance);
  const workspace = useSelectedWorkspace();

  return (
    <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
      <ScreenHeader title="Your settings" className="pt-2" />
      <View className="gap-6 px-4">
        <ProfileSection />
        <SettingsCard title="This phone">
          <SettingsRow first icon={SunMoon} label="Appearance" meta={appearanceLabel[appearance]} onPress={() => router.push("/more/settings/appearance")} />
          <SettingsRow icon={MonitorSmartphone} label="Devices" onPress={() => router.push("/more/settings/devices")} />
          <SettingsRow icon={Building2} label="Workspace" meta={workspace?.name ?? ""} onPress={() => router.push("/more/settings/workspace")} />
        </SettingsCard>
        <SignOutButton />
      </View>
    </ScrollView>
  );
};
