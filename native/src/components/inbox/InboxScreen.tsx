import { useNavigation, useRouter, type Href } from "expo-router";
import { useEffect } from "react";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { NotificationFeed } from "@/components/inbox/NotificationFeed";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useFetchNotifications, useMarkAllNotificationsRead, useMarkNotificationRead } from "@/hooks/NotificationHooks";
import { SubjectType, type Notification } from "@/models/Notification";

// A memory has no phone screen yet (ticket 11's scope); its notifications mark read with nothing to open.
// Exported so a tapped push notification can land on the same screen, without duplicating the mapping.
export const subjectRoute = (notification: Notification): Href | null => {
  if (notification.subject_type === SubjectType.Ticket) return `/board/ticket/${notification.subject_id}`;
  if (notification.subject_type === SubjectType.Doc) return `/more/docs/${notification.subject_id}`;
  return null;
};

export const InboxScreen = () => {
  const navigation = useNavigation();
  const router = useRouter();
  const { data, error, isPending, isRefetching, refetch } = useFetchNotifications();
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  useEffect(() => {
    navigation.setOptions({
      headerRight: () =>
        data && data.length > 0 ? (
          <Button variant="ghost" size="sm" disabled={markAllRead.isPending} onPress={() => markAllRead.mutate()}>
            <Text>Mark all read</Text>
          </Button>
        ) : undefined,
    });
  }, [navigation, data, markAllRead]);

  const onSelect = (notification: Notification) => {
    if (!notification.read) markRead.mutate(notification.id);
    const target = subjectRoute(notification);
    if (target) router.push(target);
  };

  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <PlaceholderScreen message="No notifications yet." />}
      {data && data.length > 0 && (
        <NotificationFeed
          notifications={data}
          refreshing={isRefetching}
          onRefresh={() => void refetch()}
          onSelect={onSelect}
        />
      )}
    </View>
  );
};
