import { Stack } from "expo-router";
import { useEffect } from "react";

import { VersionGate } from "@/components/connect/VersionGate";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { sheetOptions } from "@/lib/sheetOptions";
import { startClock } from "@/stores/clockStore";
import { useSessionStore } from "@/stores/sessionStore";

export const RootNavigator = () => {
  const signedIn = useSessionStore((s) => s.signedIn);
  useFetchMe(signedIn);
  useLiveEvents();
  useEffect(startClock, []);

  return (
    <VersionGate>
      {/* Sheets sit on this stack, above the tabs, so the dimmed backdrop covers the tab bar too. */}
      <Stack screenOptions={{ headerShown: false }}>
        <Stack.Protected guard={signedIn}>
          <Stack.Screen name="index" />
          <Stack.Screen name="(tabs)" />
          <Stack.Screen name="(sheets)/board/project-picker" options={sheetOptions} />
          <Stack.Screen name="(sheets)/board/status-picker" options={sheetOptions} />
          <Stack.Screen name="(sheets)/deploys/redeploy" options={sheetOptions} />
          <Stack.Screen name="(sheets)/more/docs/pick-project" options={sheetOptions} />
          <Stack.Screen name="(sheets)/more/settings/appearance" options={sheetOptions} />
          <Stack.Screen name="(sheets)/more/settings/workspace" options={sheetOptions} />
          <Stack.Screen name="(sheets)/more/settings/sign-out" options={sheetOptions} />
        </Stack.Protected>
        <Stack.Protected guard={!signedIn}>
          <Stack.Screen name="connect" />
        </Stack.Protected>
      </Stack>
    </VersionGate>
  );
};
