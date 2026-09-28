import * as Application from "expo-application";
import { useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { SettingsRow } from "@/components/settings/SettingsRow";
import { Text } from "@/components/ui/text";
import { useSessionStore } from "@/stores/sessionStore";

// Docs and Runners land under this same tab as siblings build their own route folders; this screen only links by path.
export const MoreScreen = () => {
  const router = useRouter();
  const host = useSessionStore((s) => s.host);

  return (
    <ScrollView className="flex-1 bg-background">
      <View className="mt-2">
        <SettingsRow label="Docs" onPress={() => router.push("/more/docs")} />
        <SettingsRow label="Runners" onPress={() => router.push("/more/runners")} />
        <SettingsRow label="Your settings" onPress={() => router.push("/more/settings")} />
      </View>
      <View className="items-center gap-1 px-4 py-6">
        {host && (
          <Text variant="muted" numberOfLines={1} className="font-mono text-xs">
            {host}
          </Text>
        )}
        <Text variant="muted" className="font-mono text-xs">
          v{Application.nativeApplicationVersion ?? "dev"}
        </Text>
      </View>
    </ScrollView>
  );
};
