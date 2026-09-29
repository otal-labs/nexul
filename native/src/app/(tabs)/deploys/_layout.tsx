import { Stack } from "expo-router";

import { sheetOptions } from "@/lib/sheetOptions";

export const unstable_settings = { initialRouteName: "index" };

export default function DeploysLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Deploys" }} />
      <Stack.Screen name="stack/[id]" options={{ title: "" }} />
      <Stack.Screen name="redeploy" options={sheetOptions} />
      <Stack.Screen name="deploy/[id]" options={{ title: "Deploy" }} />
    </Stack>
  );
}
