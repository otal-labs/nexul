import { KeyRoundIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Notification } from "@/models/Notification";

// The notice's title carries the command that sets the new token; the button opens the page that shows the state.
export const SecretsManagerNoticeDetail = ({ notification }: { notification: Notification }) => {
  const wsPath = useWorkspacePath();
  const source = notification.subject_id.split(":")[0] ?? "";
  const targets: Record<string, { to: string; label: string }> = {
    runner: { to: wsPath("/runners"), label: "Open runners" },
    automations_host: { to: wsPath("/automations/hosts"), label: "Open automations hosts" },
    server: { to: "/settings/connectors", label: "Open connectors" },
  };
  const target = targets[source] ?? targets.server;
  return (
    <EmptyState
      icon={KeyRoundIcon}
      title={notification.subject_title}
      message="References to its secrets can't be resolved until a token the manager accepts is set."
      action={
        target && (
          <Button asChild size="sm">
            <Link to={target.to}>{target.label}</Link>
          </Button>
        )
      }
    />
  );
};
