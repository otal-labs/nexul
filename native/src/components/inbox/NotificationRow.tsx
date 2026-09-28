import { Pressable, View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { NotificationKind, type Notification } from "@/models/Notification";
import { formatRelativeTime } from "@/lib/time";

const kindLabels: Record<NotificationKind, string> = {
  [NotificationKind.TicketAssigned]: "assigned to you",
  [NotificationKind.TicketMentioned]: "mentioned you",
  [NotificationKind.TicketStatusChanged]: "status changed",
  [NotificationKind.DocCreated]: "doc created",
  [NotificationKind.DocUpdated]: "doc updated",
  [NotificationKind.MemoryUpdated]: "memory updated",
  [NotificationKind.PlayRunFinished]: "play run ended",
  [NotificationKind.PlayRunWaiting]: "needs your answer",
};

interface NotificationRowProps {
  notification: Notification;
  onPress: (notification: Notification) => void;
}

export const NotificationRow = ({ notification, onPress }: NotificationRowProps) => (
  <Pressable
    role="button"
    onPress={() => onPress(notification)}
    className="min-h-11 flex-row items-start gap-2.5 border-b border-border px-4 py-3 active:bg-accent"
  >
    {!notification.read && (
      <View accessible accessibilityLabel="Unread" className="mt-1.5 size-2 shrink-0 rounded-full bg-primary" />
    )}
    <View className={cn("min-w-0 flex-1", notification.read && "pl-[14px]")}>
      <Text numberOfLines={1} className={cn(!notification.read ? "font-semibold" : "font-medium text-muted-foreground")}>
        {notification.subject_title}
      </Text>
      <Text variant="muted" numberOfLines={1} className="text-xs">
        {kindLabels[notification.kind]}
      </Text>
    </View>
    <Text variant="muted" className="font-mono text-xs">
      {formatRelativeTime(notification.created_at)}
    </Text>
  </Pressable>
);
