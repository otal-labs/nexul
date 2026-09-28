import { Stack } from "expo-router";

import { VersionGate } from "@/components/connect/VersionGate";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { useSessionStore } from "@/stores/sessionStore";

export const RootNavigator = () => {
  const signedIn = useSessionStore((s) => s.signedIn);
  useFetchMe(signedIn);
  useLiveEvents();

  return (
    <VersionGate>
      <Stack screenOptions={{ headerShown: false }}>
        <Stack.Protected guard={signedIn}>
          <Stack.Screen name="index" />
          <Stack.Screen name="(tabs)" />
        </Stack.Protected>
        <Stack.Protected guard={!signedIn}>
          <Stack.Screen name="connect" />
        </Stack.Protected>
      </Stack>
    </VersionGate>
  );
};
