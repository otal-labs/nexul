import { Stack } from "expo-router";

import { AreaGate } from "@/components/AreaGate";

export const unstable_settings = { initialRouteName: "index" };

export default function DeploysLayout() {
  return (
    <AreaGate area="stacks">
      <Stack
        screenOptions={{
          headerShadowVisible: false,
          headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
        }}
      >
        <Stack.Screen name="index" options={{ title: "Deploys" }} />
        <Stack.Screen name="stack/[id]" options={{ title: "" }} />
        <Stack.Screen name="stack/[id]/logs/[service]" options={{ title: "Logs" }} />
        <Stack.Screen name="deploy/[id]" options={{ title: "Deploy" }} />
      </Stack>
    </AreaGate>
  );
}
