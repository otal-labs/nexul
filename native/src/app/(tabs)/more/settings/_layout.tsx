import { Stack } from "expo-router";

import { sheetOptions } from "@/lib/sheetOptions";

export const unstable_settings = { initialRouteName: "index" };

export default function SettingsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Your settings" }} />
      <Stack.Screen name="devices" options={{ title: "Devices" }} />
      <Stack.Screen name="appearance" options={sheetOptions} />
      <Stack.Screen name="workspace" options={sheetOptions} />
    </Stack>
  );
}
