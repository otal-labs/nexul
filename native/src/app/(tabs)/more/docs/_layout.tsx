import { Stack } from "expo-router";

import { sheetOptions } from "@/lib/sheetOptions";

export const unstable_settings = { initialRouteName: "index" };

export default function DocsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Docs" }} />
      <Stack.Screen name="[id]" options={{ title: "" }} />
      <Stack.Screen name="pick-project" options={sheetOptions} />
    </Stack>
  );
}
