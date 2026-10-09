import { useRouter } from "expo-router";
import { ScrollView, View } from "react-native";

import { EmptyRow } from "@/components/EmptyRow";
import { ScreenHeader } from "@/components/ScreenHeader";
import { SettingsCard } from "@/components/SettingsCard";
import { DeviceRow, deviceLabel } from "@/components/settings/DeviceRow";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import type { Session } from "@/models/User";

interface DevicesFeedProps {
  sessions: Session[];
}

// This phone first, then the rest; signing out everywhere else is an outline in the destructive hue, confirmed in a sheet.
export const DevicesFeed = ({ sessions }: DevicesFeedProps) => {
  const router = useRouter();
  const current = sessions.filter((session) => session.current);
  const others = sessions.filter((session) => !session.current);

  return (
    <ScrollView className="flex-1 bg-background" contentContainerClassName="pb-8">
      <ScreenHeader title="Devices" meta="Where you're signed in" className="pt-2" />
      <View className="gap-6 px-4">
        <SettingsCard>
          {current.map((session, i) => (
            <DeviceRow key={session.id} session={session} first={i === 0} />
          ))}
        </SettingsCard>
        <View className="gap-2">
          {others.length === 0 && <EmptyRow message="No other devices are signed in." />}
          {others.length > 0 && (
            <SettingsCard title="Other devices">
              {others.map((session, i) => (
                <DeviceRow
                  key={session.id}
                  session={session}
                  first={i === 0}
                  onSignOut={() =>
                    router.push({ pathname: "/more/settings/sign-out", params: { mode: "device", id: session.id, label: deviceLabel(session) } })
                  }
                />
              ))}
            </SettingsCard>
          )}
        </View>
        <View className="gap-2">
          <Text className="text-[13px] text-muted-foreground">Everything except this phone will need to sign in again.</Text>
          <Button
            variant="outline"
            className="border-destructive/40"
            disabled={others.length === 0}
            onPress={() => router.push({ pathname: "/more/settings/sign-out", params: { mode: "others" } })}
          >
            <Text className="text-destructive">Sign out everywhere else</Text>
          </Button>
        </View>
      </View>
    </ScrollView>
  );
};
