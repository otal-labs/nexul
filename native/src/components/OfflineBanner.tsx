import { onlineManager } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";
import { View } from "react-native";

import { Text } from "@/components/ui/text";

// Fixed so the screens above the tab bar can reserve exactly this much while the banner shows.
export const OFFLINE_BANNER_HEIGHT = 36;

const subscribe = (onChange: () => void) => onlineManager.subscribe(onChange);
const getSnapshot = () => onlineManager.isOnline();

export const useIsOffline = () => !useSyncExternalStore(subscribe, getSnapshot);

export const OfflineBanner = () => {
  const offline = useIsOffline();

  if (!offline) return null;

  return (
    <View
      accessible
      role="alert"
      style={{ height: OFFLINE_BANNER_HEIGHT }}
      className="flex-row items-center gap-2 border-b border-border bg-surface-2 px-4"
    >
      <View className="size-2 rounded-full bg-warning" />
      <Text variant="small" numberOfLines={1} className="flex-1 text-muted-foreground">
        Offline. Retrying when the connection returns.
      </Text>
    </View>
  );
};
