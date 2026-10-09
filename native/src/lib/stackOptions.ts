import type { NativeStackNavigationOptions } from "expo-router/native-stack";

// Every stack's bar: flat on the canvas, the back arrow and a short Inter title; a screen's own title sits in its content.
export const stackOptions: NativeStackNavigationOptions = {
  headerShadowVisible: false,
  headerTitleStyle: { fontFamily: "Inter", fontWeight: "600", fontSize: 16 },
};

// A tab's root draws its own header (ScreenHeader) under the status bar.
export const tabRootOptions: NativeStackNavigationOptions = { headerShown: false };
