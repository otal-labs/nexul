import { FlatList, RefreshControl } from "react-native";
import { useCSSVariable } from "uniwind";

import { NotificationRow } from "@/components/inbox/NotificationRow";
import type { Notification } from "@/models/Notification";

interface NotificationFeedProps {
  notifications: Notification[];
  refreshing: boolean;
  onRefresh: () => void;
  onSelect: (notification: Notification) => void;
}

// Order is the server's (newest first, ORDER BY created_at DESC); the feed renders it as given.
export const NotificationFeed = ({ notifications, refreshing, onRefresh, onSelect }: NotificationFeedProps) => {
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <FlatList
      data={notifications}
      keyExtractor={(notification) => notification.id}
      renderItem={({ item }) => <NotificationRow notification={item} onPress={onSelect} />}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={String(mutedForeground)} />
      }
    />
  );
};
