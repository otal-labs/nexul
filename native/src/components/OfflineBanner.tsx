import { onlineManager } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";
import { View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { Text } from "@/components/ui/text";

const subscribe = (onChange: () => void) => onlineManager.subscribe(onChange);
const getSnapshot = () => onlineManager.isOnline();

export const OfflineBanner = () => {
  const online = useSyncExternalStore(subscribe, getSnapshot);
  const { top } = useSafeAreaInsets();

  if (online) return null;

  return (
    <View
      accessible
      role="alert"
      style={{ paddingTop: top }}
      className="flex-row items-center gap-2 border-b border-border bg-surface-2 px-4 pb-2"
    >
      <View className="size-2 rounded-full bg-warning" />
      <Text variant="small" className="text-muted-foreground">
        Offline. Retrying when the connection returns.
      </Text>
    </View>
  );
};
