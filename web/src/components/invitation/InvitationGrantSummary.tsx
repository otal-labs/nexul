import { RoleChip } from "@/components/role/RoleChip";
import type { InvitationPreview } from "@/models/Invitation";

// Each workspace the link admits to, with every role it gives there as chips.
export const InvitationGrantSummary = ({ invitation }: { invitation: InvitationPreview }) => (
  <ul className="divide-y divide-border overflow-hidden rounded-md border bg-card text-sm">
    {invitation.grants.map((grant) => (
      <li key={grant.workspace_id} className="flex flex-wrap items-center justify-between gap-2 px-3 py-3">
        <span className="font-medium">{grant.workspace_name}</span>
        <span className="flex flex-wrap justify-end gap-1">
          {grant.roles.map((role) => (
            <RoleChip key={role.id} name={role.name} />
          ))}
        </span>
      </li>
    ))}
  </ul>
);
