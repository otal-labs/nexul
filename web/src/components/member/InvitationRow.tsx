import { Link2Off } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import type { ActiveInvitation } from "@/models/Invitation";
import { daysUntil } from "@/utils/TimeUtility";

interface InvitationRowProps {
  invitation: ActiveInvitation;
  inviter: string;
  onRevoke: (id: string) => void;
  disabled: boolean;
}

const expiresLabel = (expiresAt: string): string => {
  const days = daysUntil(expiresAt);
  if (days <= 1) return "expires within a day";
  return `expires in ${days} days`;
};

const grantsLabel = (invitation: ActiveInvitation): string =>
  invitation.grants.map((grant) => `${grant.workspace_name} · ${grant.role_name}`).join(", ");

export const InvitationRow = ({ invitation, inviter, onRevoke, disabled }: InvitationRowProps) => (
  <li className="flex items-center gap-3 px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40 sm:px-4">
    <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground"><Link2Off className="size-4" aria-hidden /></span>
    <div className="min-w-0 flex-1">
      <p className="truncate text-sm font-medium" title={grantsLabel(invitation)}>{grantsLabel(invitation)}</p>
      <p className="truncate text-xs text-muted-foreground">by {inviter} · {expiresLabel(invitation.expires_at)}</p>
    </div>
    <ConfirmDestroyButton icon={Link2Off} idleLabel="Revoke invitation" onConfirm={() => onRevoke(invitation.id)} disabled={disabled} />
  </li>
);
