import "@/global.css";

import { QueryClientProvider, focusManager } from "@tanstack/react-query";
import { ThemeProvider } from "expo-router/react-navigation";
import { StatusBar } from "expo-status-bar";
import { useEffect } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { RootNavigator } from "@/components/RootNavigator";
import { useNavigationTheme } from "@/hooks/useNavigationTheme";
import { queryClient } from "@/lib/queryClient";

export { ErrorBoundary } from "expo-router";

const onAppStateChange = (status: AppStateStatus) => focusManager.setFocused(status === "active");

export default function RootLayout() {
  const navigationTheme = useNavigationTheme();

  useEffect(() => {
    const subscription = AppState.addEventListener("change", onAppStateChange);
    return () => subscription.remove();
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider value={navigationTheme}>
        <StatusBar style="auto" />
        <RootNavigator />
      </ThemeProvider>
    </QueryClientProvider>
  );
}
