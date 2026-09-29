import { Stack } from "expo-router";

export const unstable_settings = { initialRouteName: "index" };

// docs/ and settings/ are stacks of their own that draw their own header, so this stack hides its bar over them.
export default function MoreLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "More" }} />
      <Stack.Screen name="runners/index" options={{ title: "Runners" }} />
      <Stack.Screen name="docs" options={{ headerShown: false }} />
      <Stack.Screen name="settings" options={{ headerShown: false }} />
    </Stack>
  );
}
