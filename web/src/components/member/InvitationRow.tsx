import { Link2Off } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import type { ActiveInvitation } from "@/models/Invitation";

interface InvitationRowProps {
  invitation: ActiveInvitation;
  onRevoke: (id: string) => void;
  disabled: boolean;
}

export const InvitationRow = ({ invitation, onRevoke, disabled }: InvitationRowProps) => (
  <li className="flex flex-wrap items-center gap-3 px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40 sm:px-4">
    <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground"><Link2Off className="size-4" aria-hidden /></span>
    <div className="min-w-0 flex-1">
      <p className="font-mono text-sm font-medium">Invitation {invitation.id.slice(0, 8)}</p>
      <p className="text-xs text-muted-foreground">{invitation.grants.map((grant) => `${grant.workspace_name} · ${grant.role_name}`).join(" · ")}</p>
      <p className="font-mono text-xs text-muted-foreground">expires {new Date(invitation.expires_at).toLocaleString()}</p>
    </div>
    <ConfirmDestroyButton icon={Link2Off} idleLabel="Revoke invitation" onConfirm={() => onRevoke(invitation.id)} disabled={disabled} />
  </li>
);
