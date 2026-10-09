import { Stack } from "expo-router";

import { stackOptions } from "@/lib/stackOptions";

export const ConnectStack = () => (
  <Stack
    screenOptions={stackOptions}
  >
    <Stack.Screen name="index" options={{ headerShown: false }} />
    <Stack.Screen name="scan" options={{ title: "Scan the code" }} />
    <Stack.Screen name="manual" options={{ title: "Enter it by hand" }} />
  </Stack>
);
