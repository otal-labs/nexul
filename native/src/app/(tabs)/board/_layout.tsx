import { Stack } from "expo-router";

export default function BoardLayout() {
  return (
    <Stack
      screenOptions={{
        headerShadowVisible: false,
        headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
      }}
    >
      <Stack.Screen name="index" options={{ title: "Board" }} />
      <Stack.Screen name="project-picker" options={{ title: "Choose a project", presentation: "formSheet" }} />
      <Stack.Screen name="status-picker" options={{ title: "Change status", presentation: "formSheet" }} />
      <Stack.Screen name="ticket/[id]" options={{ title: "Ticket" }} />
    </Stack>
  );
}
