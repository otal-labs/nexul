import { Stack } from "expo-router";

export const unstable_settings = { initialRouteName: "index" };

export default function BoardLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Board" }} />
      <Stack.Screen name="ticket/[id]" options={{ title: "" }} />
    </Stack>
  );
}
