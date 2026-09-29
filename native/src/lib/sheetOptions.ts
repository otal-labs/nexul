import type { NativeStackNavigationOptions } from "expo-router/native-stack";

// Header hidden: Android sheets never show it, and a live one crashes react-native-screens on a theme change mid-dismiss.
export const sheetOptions: NativeStackNavigationOptions = {
  presentation: "formSheet",
  headerShown: false,
  sheetAllowedDetents: "fitToContents",
};
