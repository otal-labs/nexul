import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { TeamGapLine } from "@/components/team/TeamGapLine";
import { invitationGrantsLabel, type ActiveInvitation } from "@/models/Invitation";
import type { TeamPerson } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AcceptedInvitationRowProps {
  invitation: ActiveInvitation;
  person: TeamPerson;
  inviter: string;
  onOpen: (id: string) => void;
}

// Who a link let in, what it gave them, and how far their setup got; opens the person.
export const AcceptedInvitationRow = ({ invitation, person, inviter, onOpen }: AcceptedInvitationRowProps) => {
  const granted = invitationGrantsLabel(invitation);
  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(person.id)}
        aria-label={`Open ${personName(person)}`}
        className="flex w-full items-center gap-3 px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent sm:px-4"
      >
        <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-7 shrink-0" />
        <span className="min-w-0 flex-1 space-y-1">
          <span dir="auto" className="block truncate text-sm font-medium" title={personName(person)}>
            {personName(person)}
          </span>
          <span className="block truncate text-xs text-muted-foreground">
            {granted && `${granted} · `}invited by {inviter}
          </span>
          <TeamGapLine person={person} />
        </span>
        {invitation.redeemed_at && (
          <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">accepted {formatRelativeTime(invitation.redeemed_at)}</span>
        )}
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </button>
    </li>
  );
};
