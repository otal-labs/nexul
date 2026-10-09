import type { NativeStackNavigationOptions } from "expo-router/native-stack";

// Header hidden: Android sheets never show it, and a live one crashes react-native-screens on a theme change mid-dismiss.
export const sheetOptions: NativeStackNavigationOptions = {
  presentation: "formSheet",
  headerShown: false,
  sheetAllowedDetents: "fitToContents",
  // The panel's 12pt corner; the sheet itself rises and swipes away with the platform's own motion.
  sheetCornerRadius: 12,
  sheetGrabberVisible: true,
};
