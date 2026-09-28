import { Stack } from "expo-router";

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
      <Stack.Screen name="appearance" options={{ title: "Appearance", presentation: "formSheet" }} />
      <Stack.Screen name="workspace" options={{ title: "Switch workspace", presentation: "formSheet" }} />
    </Stack>
  );
}
