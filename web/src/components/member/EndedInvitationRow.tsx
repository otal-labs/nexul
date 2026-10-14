import { Link2Off } from "lucide-react";

import { invitationLinkLabel, type ActiveInvitation } from "@/models/Invitation";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface EndedInvitationRowProps {
  invitation: ActiveInvitation;
  inviter: string;
}

// A link that expired or was deleted before anyone used it: what it gave, who made it and when, and when it ended.
export const EndedInvitationRow = ({ invitation, inviter }: EndedInvitationRowProps) => {
  const label = invitationLinkLabel(invitation) || "A link whose roles are gone";
  return (
    <li className="flex items-center gap-3 px-3 py-3 sm:px-4">
      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <Link2Off className="size-4" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-muted-foreground" title={label}>
          {label}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          by {inviter} · made {formatRelativeTime(invitation.created_at)}
        </p>
      </div>
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">ended {formatRelativeTime(invitation.expires_at)}</span>
    </li>
  );
};
