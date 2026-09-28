import { Stack } from "expo-router";

// Not <TabStack>: the nested settings/ folder needs its outer screen header hidden so its own stack owns the header.
export default function MoreLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "More" }} />
      <Stack.Screen name="settings" options={{ headerShown: false }} />
    </Stack>
  );
}
