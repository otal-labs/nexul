import { Stack } from "expo-router";

import { stackOptions } from "@/lib/stackOptions";

export const unstable_settings = { initialRouteName: "index" };

export default function DocsLayout() {
  return (
    <Stack
      screenOptions={stackOptions}
    >
      <Stack.Screen name="index" options={{ title: "" }} />
      <Stack.Screen name="[id]" options={{ title: "" }} />
    </Stack>
  );
}
