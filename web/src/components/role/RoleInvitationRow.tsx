import { Link2 } from "lucide-react";

import { RoleChipRow } from "@/components/role/RoleChipRow";
import { rowClass } from "@/components/role/rowStyles";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { ActiveInvitation } from "@/models/Invitation";
import { daysUntil } from "@/utils/TimeUtility";

interface RoleInvitationRowProps {
  invitation: ActiveInvitation;
  workspaceId: string;
  // The open role's name, which leads the title.
  openName: string;
  inviter: string;
  onRevoke: (id: string) => void;
}

const expires = (at: string): string => {
  const days = daysUntil(at);
  return days <= 1 ? "expires within a day" : `expires in ${days} days`;
};

// An open link on the holders' grid: the link mark where a picture sits, who made it and when it expires, the other roles it
// gives as chips, and its menu.
export const RoleInvitationRow = ({ invitation, workspaceId, openName, inviter, onRevoke }: RoleInvitationRowProps) => {
  const { open: confirm } = useConfirmationDialog();
  // What else the link gives besides the open role, as the holders show their other roles.
  const others = invitation.grants.flatMap((grant) =>
    grant.roles
      .filter((role) => grant.workspace_id !== workspaceId || role.name !== openName)
      .map((role) => ({ id: `${grant.workspace_id}:${role.id}`, name: grant.workspace_id === workspaceId ? role.name : `${role.name} in ${grant.workspace_name}`, is_owner: false })),
  );
  // Titled by when it stops working, which is what tells two links here apart.
  const title = `Invite link, ${expires(invitation.expires_at)}`;
  const revoke = async () => {
    const ok = await confirm({ title: "Revoke this invitation link?", message: "Whoever has it can no longer use it to sign up.", confirmLabel: "Revoke link" });
    if (ok) onRevoke(invitation.id);
  };

  return (
    <li className="flex min-h-14 border-b border-border py-1">
      <div className={rowClass}>
        <span className="flex size-7 shrink-0 items-center justify-center rounded-full border border-border bg-surface-2 text-muted-foreground">
          <Link2 aria-hidden className="size-3.5" />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{title}</span>
          <span className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
            <span className="shrink-0">By {inviter}</span>
            {others.length > 0 && (
              <span className="-my-0.5 flex min-w-0 flex-1">
                <RoleChipRow roles={others} />
              </span>
            )}
          </span>
        </span>
        <RowActionsMenu subject={`${title}, by ${inviter}`} actions={[{ label: "Revoke link", destructive: true, permission: "members:write", verb: "revoke invitations", onSelect: () => void revoke() }]} />
      </div>
    </li>
  );
};
