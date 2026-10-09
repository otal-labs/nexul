import { ScreenHeader } from "@/components/ScreenHeader";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useMarkAllNotificationsRead } from "@/hooks/NotificationHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { Notification } from "@/models/Notification";

interface InboxHeaderProps {
  notifications: Notification[] | undefined;
}

const unreadLine = (count: number) => (count === 0 ? "All read" : `${count} unread`);

export const InboxHeader = ({ notifications }: InboxHeaderProps) => {
  const workspace = useSelectedWorkspace();
  const markAllRead = useMarkAllNotificationsRead();
  const unread = notifications?.filter((n) => !n.read).length ?? 0;
  return (
    <ScreenHeader
      eyebrow={workspace?.name}
      title="Inbox"
      meta={notifications && notifications.length > 0 ? unreadLine(unread) : undefined}
      action={
        unread > 0 && (
          <Button variant="ghost" size="sm" disabled={markAllRead.isPending} onPress={() => markAllRead.mutate()}>
            <Text className="text-sm text-muted-foreground">Mark all read</Text>
          </Button>
        )
      }
    />
  );
};
