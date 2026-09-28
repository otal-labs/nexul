import { Alert, View } from "react-native";

import { DeviceRow } from "@/components/settings/DeviceRow";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useSignOutOtherSessions, useSignOutSession } from "@/hooks/SessionHooks";
import type { Session } from "@/models/User";

interface DevicesFeedProps {
  sessions: Session[];
}

// Current device first, then the rest; nothing animates on the phone (ticket 12), unlike the web version's arrival/leaving motion.
export const DevicesFeed = ({ sessions }: DevicesFeedProps) => {
  const signOutOne = useSignOutSession();
  const signOutOthers = useSignOutOtherSessions();
  const current = sessions.filter((session) => session.current);
  const others = sessions.filter((session) => !session.current);

  const confirmSignOutAll = () =>
    Alert.alert("Sign out everywhere else?", "Every other device will need to sign in again.", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign out", style: "destructive", onPress: () => signOutOthers.mutate() },
    ]);

  return (
    <View className="flex-1 bg-background">
      {current.map((session) => (
        <DeviceRow key={session.id} session={session} />
      ))}
      <Text variant="muted" className="px-4 pt-4 pb-1 text-xs">
        Other devices
      </Text>
      {others.length === 0 && (
        <Text variant="muted" className="px-4 py-3 text-sm">
          No other devices are signed in.
        </Text>
      )}
      {others.map((session) => (
        <DeviceRow
          key={session.id}
          session={session}
          pending={signOutOne.isPending && signOutOne.variables === session.id}
          onSignOut={() => signOutOne.mutate(session.id)}
        />
      ))}
      <View className="gap-2 px-4 py-4">
        <Text variant="muted" className="text-sm">
          Everything except this device will need to sign in again.
        </Text>
        <Button variant="destructive" size="sm" disabled={others.length === 0 || signOutOthers.isPending} onPress={confirmSignOutAll}>
          <Text>Sign out everywhere else</Text>
        </Button>
      </View>
    </View>
  );
};
