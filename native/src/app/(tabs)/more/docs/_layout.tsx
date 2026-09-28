import { Stack } from "expo-router";

export default function DocsLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Docs" }} />
      <Stack.Screen name="[id]" options={{ title: "Doc" }} />
      <Stack.Screen name="pick-project" options={{ title: "Choose a project", presentation: "formSheet" }} />
    </Stack>
  );
}
