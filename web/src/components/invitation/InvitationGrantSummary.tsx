import type { InvitationAcceptance } from "@/models/Invitation";

interface InvitationGrantSummaryProps {
  invitation: InvitationAcceptance;
  detailed?: boolean;
}

export const InvitationGrantSummary = ({ invitation, detailed = false }: InvitationGrantSummaryProps) => (
  <ul className="divide-y divide-border overflow-hidden rounded-md border bg-card text-sm">
    {invitation.grants.map((grant) => (
      <li key={grant.workspace_id} className="flex flex-wrap items-baseline justify-between gap-2 px-3 py-3">
        <span className="font-medium">{grant.workspace_name}</span>
        <div className="text-right">
          <span className="font-mono text-xs text-muted-foreground">{grant.role_name}</span>
          {detailed && (
            <div className="mt-1 space-y-0.5 text-[11px] text-muted-foreground">
              {grant.allow.length > 0 && <p>Allow: {grant.allow.join(", ")}</p>}
              {grant.deny.length > 0 && <p>Deny: {grant.deny.join(", ")}</p>}
              {grant.allow.length === 0 && grant.deny.length === 0 && <p>No overrides</p>}
            </div>
          )}
        </div>
      </li>
    ))}
  </ul>
);
