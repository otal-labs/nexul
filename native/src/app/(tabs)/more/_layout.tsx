import { Stack } from "expo-router";

import { stackOptions, tabRootOptions } from "@/lib/stackOptions";

export const unstable_settings = { initialRouteName: "index" };

// calendar/, docs/ and settings/ are stacks of their own that draw their own header, so this stack hides its bar over them.
export default function MoreLayout() {
  return (
    <Stack
      screenOptions={stackOptions}
    >
      <Stack.Screen name="index" options={{ ...tabRootOptions, title: "More" }} />
      <Stack.Screen name="search" options={{ title: "Search" }} />
      <Stack.Screen name="runners/index" options={{ title: "" }} />
      <Stack.Screen name="calendar" options={{ headerShown: false }} />
      <Stack.Screen name="docs" options={{ headerShown: false }} />
      <Stack.Screen name="settings" options={{ headerShown: false }} />
    </Stack>
  );
}
