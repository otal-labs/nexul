import { View } from "react-native";
import { BottomTabBar, type BottomTabBarProps } from "expo-router/js-tabs";

import { OfflineBanner } from "@/components/OfflineBanner";

// The banner floats over the screens' bottom edge; the tab layout pads the scenes by its height while it shows.
export const TabBar = (props: BottomTabBarProps) => (
  <View>
    <View className="absolute inset-x-0 bottom-full">
      <OfflineBanner />
    </View>
    <BottomTabBar {...props} />
  </View>
);
