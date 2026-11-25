import * as Application from "expo-application";
import { useRouter } from "expo-router";
import CalendarDays from "lucide-react-native/icons/calendar-days";
import FileText from "lucide-react-native/icons/file-text";
import Search from "lucide-react-native/icons/search";
import Server from "lucide-react-native/icons/server";
import Settings from "lucide-react-native/icons/settings";
import Zap from "lucide-react-native/icons/zap";
import { ScrollView, View } from "react-native";

import { ScreenHeader } from "@/components/ScreenHeader";
import { SettingsCard } from "@/components/SettingsCard";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { FieldScreen } from "@/components/FieldScreen";
import { Text } from "@/components/ui/text";
import { useShowRunnersRow } from "@/hooks/RunnerHooks";
import { useAreaAccess, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useSessionStore } from "@/stores/sessionStore";

// Docs, Plays and Runners land under this same tab as siblings build their own route folders; this screen only links by path.
export const MoreScreen = () => {
  const router = useRouter();
  const host = useSessionStore((s) => s.host);
  const workspace = useSelectedWorkspace();
  const showRunners = useShowRunnersRow();
  const showPlays = useAreaAccess()?.("plays") === true;

  return (
    <FieldScreen>
      <ScrollView className="flex-1" contentContainerClassName="pb-6">
        <ScreenHeader eyebrow={workspace?.name} title="More" />
        <View className="gap-6 px-4">
          <SettingsCard>
            <SettingsRow first icon={Search} label="Search" onPress={() => router.push("/more/search")} />
            <SettingsRow icon={CalendarDays} label="Calendar" onPress={() => router.push("/more/calendar")} />
            <SettingsRow icon={FileText} label="Docs" onPress={() => router.push("/more/docs")} />
            {showPlays && <SettingsRow icon={Zap} label="Plays" onPress={() => router.push("/more/plays")} />}
            {showRunners && <SettingsRow icon={Server} label="Runners" onPress={() => router.push("/more/runners")} />}
          </SettingsCard>
          <SettingsCard>
            <SettingsRow first icon={Settings} label="Your settings" onPress={() => router.push("/more/settings")} />
          </SettingsCard>
          <View className="items-center gap-1 pt-2">
            {host && (
              <Text numberOfLines={1} className="font-mono text-xs text-muted-foreground">
                {host}
              </Text>
            )}
            <Text className="font-mono text-xs text-muted-foreground">v{Application.nativeApplicationVersion ?? "dev"}</Text>
          </View>
        </View>
      </ScrollView>
    </FieldScreen>
  );
};
