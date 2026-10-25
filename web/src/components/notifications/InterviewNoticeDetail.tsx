import { ClipboardListIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Notification } from "@/models/Notification";

// The Interview's tabs live in the URL, so it opens on its own page rather than in the inbox pane.
export const InterviewNoticeDetail = ({ notification }: { notification: Notification }) => {
  const wsPath = useWorkspacePath();
  return (
    <EmptyState
      icon={ClipboardListIcon}
      title={notification.subject_title}
      message="The project's Interview has the questions waiting on you."
      action={
        <Button asChild size="sm">
          <Link to={wsPath(`/projects/${notification.subject_id}/interview`)}>Open interview</Link>
        </Button>
      }
    />
  );
};
