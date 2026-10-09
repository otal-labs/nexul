import { Stack } from "expo-router";

import { stackOptions, tabRootOptions } from "@/lib/stackOptions";

import { AreaGate } from "@/components/AreaGate";

export const unstable_settings = { initialRouteName: "index" };

export default function DeploysLayout() {
  return (
    <AreaGate area="stacks">
      <Stack
        screenOptions={stackOptions}
      >
        <Stack.Screen name="index" options={{ ...tabRootOptions, title: "Deploys" }} />
        <Stack.Screen name="stack/[id]" options={{ title: "" }} />
        <Stack.Screen name="stack/[id]/logs/[service]" options={{ title: "Logs" }} />
        <Stack.Screen name="deploy/[id]" options={{ title: "Deploy" }} />
      </Stack>
    </AreaGate>
  );
}
