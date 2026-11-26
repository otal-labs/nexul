import type { ReactNode } from "react";
import { Platform, View, type ViewProps } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { cn } from "@/lib/utils";

interface SheetBodyProps extends Pick<ViewProps, "role"> {
  className?: string;
  children: ReactNode;
}

// A sheet's root: Android's form sheet draws no grabber (iOS draws its own), and the last control clears the gesture bar by 16pt.
export const SheetBody = ({ className, role, children }: SheetBodyProps) => {
  const insets = useSafeAreaInsets();
  return (
    <View role={role} className={cn("bg-popover", className)} style={{ paddingBottom: insets.bottom + 16 }}>
      {Platform.OS === "android" && <View className="mt-4 h-1 w-8 self-center rounded-full bg-muted-foreground/40" />}
      {children}
    </View>
  );
};
