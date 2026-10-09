import { useRouter, type Href } from "expo-router";
import Inbox from "lucide-react-native/icons/inbox";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HandOff, useLoaderShown } from "@/components/HandOff";
import { InboxHeader } from "@/components/inbox/InboxHeader";
import { NotificationFeed } from "@/components/inbox/NotificationFeed";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FieldScreen } from "@/components/FieldScreen";
import { useFetchNotifications, useMarkNotificationRead } from "@/hooks/NotificationHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { SubjectType, type Notification } from "@/models/Notification";

// A memory has no phone screen yet (ticket 11's scope); its notifications mark read with nothing to open.
// Exported so a tapped push notification can land on the same screen, without duplicating the mapping.
export const subjectRoute = (notification: Notification): Href | null => {
  if (notification.subject_type === SubjectType.Ticket) return `/board/ticket/${notification.subject_id}`;
  if (notification.subject_type === SubjectType.Doc) return `/more/docs/${notification.subject_id}`;
  return null;
};

export const InboxScreen = () => {
  const router = useRouter();
  const { data, error, isPending, isRefetching, refetch } = useFetchNotifications();
  const markRead = useMarkNotificationRead();
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  const waited = useLoaderShown(isPending);

  const onSelect = (notification: Notification) => {
    if (!notification.read) markRead.mutate(notification.id);
    const target = subjectRoute(notification);
    // A ticket opens on the Board tab, which a viewer without tickets:read doesn't have.
    if (notification.subject_type === SubjectType.Ticket && !canReadTickets) return;
    if (target) router.push(target, { withAnchor: true });
  };

  return (
    <FieldScreen>
      {!(data && data.length > 0) && <InboxHeader notifications={data} />}
      {isPending && <LoadingDisplay message="Loading the inbox" />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && (
        <EmptyState icon={Inbox} title="Nothing needs you" message="Assignments, mentions and play runs waiting on you land here." />
      )}
      {data && data.length > 0 && (
        <HandOff after={waited}>
          <NotificationFeed
            notifications={data}
            refreshing={isRefetching}
            onRefresh={() => void refetch()}
            onSelect={onSelect}
            onMarkRead={(notification) => markRead.mutate(notification.id)}
          />
        </HandOff>
      )}
    </FieldScreen>
  );
};
