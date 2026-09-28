import { Stack } from "expo-router";

interface TabStackProps {
  title: string;
}

export const TabStack = ({ title }: TabStackProps) => (
  <Stack
    screenOptions={{
      headerShadowVisible: false,
      headerTitleStyle: { fontFamily: "Inter", fontWeight: "600" },
    }}
  >
    <Stack.Screen name="index" options={{ title }} />
  </Stack>
);
