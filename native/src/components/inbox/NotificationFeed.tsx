import { FlatList } from "react-native";

import { InboxHeader } from "@/components/inbox/InboxHeader";
import { NotificationRow } from "@/components/inbox/NotificationRow";
import { RefreshList } from "@/components/RefreshList";
import type { Notification } from "@/models/Notification";

interface NotificationFeedProps {
  notifications: Notification[];
  refreshing: boolean;
  onRefresh: () => void;
  onSelect: (notification: Notification) => void;
  onMarkRead: (notification: Notification) => void;
}

// Order is the server's (newest first, ORDER BY created_at DESC); the feed renders it as given.
export const NotificationFeed = ({ notifications, refreshing, onRefresh, onSelect, onMarkRead }: NotificationFeedProps) => (
  <FlatList
    data={notifications}
    keyExtractor={(notification) => notification.id}
    ListHeaderComponent={<InboxHeader notifications={notifications} />}
    contentContainerClassName="pb-6"
    renderItem={({ item }) => <NotificationRow notification={item} onPress={onSelect} onMarkRead={onMarkRead} />}
    refreshControl={<RefreshList refreshing={refreshing} onRefresh={onRefresh} />}
  />
);
