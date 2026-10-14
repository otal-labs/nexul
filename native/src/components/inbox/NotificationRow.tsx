import CheckCheck from "lucide-react-native/icons/check-check";
import Circle from "lucide-react-native/icons/circle";
import Mail from "lucide-react-native/icons/mail";
import { useRef } from "react";
import { Pressable, View } from "react-native";
import Swipeable, { type SwipeableMethods } from "react-native-gesture-handler/ReanimatedSwipeable";
import Animated, { Extrapolation, interpolate, useAnimatedStyle, type SharedValue } from "react-native-reanimated";
import { useCSSVariable } from "uniwind";

import { notificationKindIcon } from "@/components/inbox/notificationKindIcon";
import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";
import { UnreadDot } from "@/components/UnreadDot";
import { cn } from "@/lib/utils";
import { NotificationKind, type Notification } from "@/models/Notification";

const kindLabels: Record<NotificationKind, string> = {
  [NotificationKind.TicketAssigned]: "Assigned to you",
  [NotificationKind.TicketMentioned]: "You were mentioned",
  [NotificationKind.TicketStatusChanged]: "Status changed",
  [NotificationKind.DocCreated]: "Doc created",
  [NotificationKind.DocUpdated]: "Doc updated",
  [NotificationKind.DocMentioned]: "You were mentioned",
  [NotificationKind.DocQuestionsAsked]: "New questions",
  [NotificationKind.DocQuestionsAnswered]: "Questions answered",
  [NotificationKind.MemoryUpdated]: "Memory updated",
  [NotificationKind.PlayRunFinished]: "Play run ended",
  [NotificationKind.PlayRunWaiting]: "Needs your answer",
  [NotificationKind.ComputerMoveRequested]: "Move to the Nexul app",
};

interface NotificationRowProps {
  notification: Notification;
  onPress: (notification: Notification) => void;
  onMarkRead: (notification: Notification) => void;
  onMarkUnread: (notification: Notification) => void;
}

// What a left swipe uncovers: the icon and word of the move it makes, growing in as the row is pulled.
const SwipeAction = ({ progress, icon: Glyph, label }: { progress: SharedValue<number>; icon: typeof CheckCheck; label: string }) => {
  const [foreground] = useCSSVariable(["--color-foreground"]);
  const style = useAnimatedStyle(() => ({
    opacity: Math.min(1, progress.get()),
    transform: [{ scale: interpolate(progress.get(), [0, 1], [0.6, 1], Extrapolation.CLAMP) }],
  }));
  return (
    <View className="w-24 items-center justify-center bg-accent">
      <Animated.View style={style} className="items-center gap-1">
        <Glyph size={20} color={String(foreground)} />
        <Text className="text-xs font-medium">{label}</Text>
      </Animated.View>
    </View>
  );
};

// A tile for what happened with the unread dot on its corner, the title with its time, and what happened under it.
// A row swipes left to flip between read and unread without opening it; the same move is its accessibility action.
export const NotificationRow = ({ notification, onPress, onMarkRead, onMarkUnread }: NotificationRowProps) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const swipeable = useRef<SwipeableMethods>(null);
  const Icon = notificationKindIcon[notification.kind] ?? Circle;
  const unread = !notification.read;
  const flip = unread ? onMarkRead : onMarkUnread;
  return (
    <Swipeable
      ref={swipeable}
      friction={1.5}
      rightThreshold={56}
      overshootRight={false}
      renderRightActions={(progress) => (
        <SwipeAction progress={progress} icon={unread ? CheckCheck : Mail} label={unread ? "Read" : "Unread"} />
      )}
      onSwipeableWillOpen={() => flip(notification)}
      onSwipeableOpen={() => swipeable.current?.close()}
    >
      <Pressable
        role="button"
        accessibilityActions={[unread ? { name: "markRead", label: "Mark read" } : { name: "markUnread", label: "Mark unread" }]}
        onAccessibilityAction={() => flip(notification)}
        onPress={() => onPress(notification)}
        className="min-h-16 flex-row items-start gap-3.5 bg-background px-5 py-3 active:bg-accent"
      >
        <View className="size-9 items-center justify-center rounded-lg border border-border bg-card">
          <Icon size={16} color={String(muted)} />
          <UnreadDot unread={unread} className="absolute -right-1 -top-1" />
        </View>
        <View className="min-w-0 flex-1 gap-0.5">
          <View className="flex-row items-baseline gap-3">
            <Text numberOfLines={2} className={cn("min-w-0 flex-1 text-[15px] leading-5", unread ? "font-medium" : "text-muted-foreground")}>
              {notification.subject_title}
            </Text>
            <Text className="font-mono text-xs text-muted-foreground">
              <RelativeTime iso={notification.created_at} />
            </Text>
          </View>
          <Text numberOfLines={1} className="text-[13px] text-muted-foreground">
            {kindLabels[notification.kind]}
          </Text>
        </View>
      </Pressable>
    </Swipeable>
  );
};
