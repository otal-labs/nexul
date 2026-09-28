import { BottomTabBar, type BottomTabBarProps } from "expo-router/js-tabs";

import { OfflineBanner } from "@/components/OfflineBanner";

// Above the tab bar the banner needs no inset math and never pushes a native header down.
export const TabBar = (props: BottomTabBarProps) => (
  <>
    <OfflineBanner />
    <BottomTabBar {...props} />
  </>
);
