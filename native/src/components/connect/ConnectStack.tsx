import { Stack } from "expo-router";

export const ConnectStack = () => (
  <Stack
    screenOptions={{
      headerShadowVisible: false,
      headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
    }}
  >
    <Stack.Screen name="index" options={{ headerShown: false }} />
    <Stack.Screen name="scan" options={{ title: "Scan the code" }} />
    <Stack.Screen name="manual" options={{ title: "Enter it by hand" }} />
  </Stack>
);
