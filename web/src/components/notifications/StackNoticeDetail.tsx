import { ServerIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Notification } from "@/models/Notification";

// A stack notice opens on what happened and where to look; the stack's page carries the outage line and the logs.
export const StackNoticeDetail = ({ notification }: { notification: Notification }) => {
  const wsPath = useWorkspacePath();
  return (
    <EmptyState
      icon={ServerIcon}
      title={notification.subject_title}
      message="The stack's page shows what is down now and, with the right to read logs, the crashed service's last output."
      action={
        <Button asChild size="sm">
          <Link to={wsPath(`/stacks/${notification.subject_id}`)}>Open stack</Link>
        </Button>
      }
    />
  );
};
