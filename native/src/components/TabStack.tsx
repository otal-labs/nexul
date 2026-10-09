import { Stack } from "expo-router";

import { stackOptions, tabRootOptions } from "@/lib/stackOptions";

interface TabStackProps {
  title: string;
}

export const TabStack = ({ title }: TabStackProps) => (
  <Stack screenOptions={stackOptions}>
    <Stack.Screen name="index" options={{ ...tabRootOptions, title }} />
  </Stack>
);
