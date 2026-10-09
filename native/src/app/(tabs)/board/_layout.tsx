import { Stack } from "expo-router";

import { stackOptions, tabRootOptions } from "@/lib/stackOptions";

import { AreaGate } from "@/components/AreaGate";

export const unstable_settings = { initialRouteName: "index" };

export default function BoardLayout() {
  return (
    <AreaGate area="tickets">
      <Stack
        screenOptions={stackOptions}
      >
        <Stack.Screen name="index" options={{ ...tabRootOptions, title: "Board" }} />
        <Stack.Screen name="ticket/[id]" options={{ title: "" }} />
      </Stack>
    </AreaGate>
  );
}
