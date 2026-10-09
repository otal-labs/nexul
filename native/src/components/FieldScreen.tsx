import type { ReactNode } from "react";
import { View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { LightField } from "@/components/LightField";

interface FieldScreenProps {
  children: ReactNode;
}

// A tab's root and the connect screens: the canvas with the still light field behind it, content below the status bar, no app bar.
export const FieldScreen = ({ children }: FieldScreenProps) => {
  const insets = useSafeAreaInsets();
  return (
    <View className="flex-1 bg-background">
      <LightField />
      <View style={{ flex: 1, paddingTop: insets.top }}>
        {children}
      </View>
    </View>
  );
};
