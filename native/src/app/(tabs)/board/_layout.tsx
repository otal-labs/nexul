import { Stack } from "expo-router";

import { AreaGate } from "@/components/AreaGate";

export const unstable_settings = { initialRouteName: "index" };

export default function BoardLayout() {
  return (
    <AreaGate area="tickets">
      <Stack
        screenOptions={{
          headerShadowVisible: false,
          headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
        }}
      >
        <Stack.Screen name="index" options={{ title: "Board" }} />
        <Stack.Screen name="ticket/[id]" options={{ title: "" }} />
      </Stack>
    </AreaGate>
  );
}
