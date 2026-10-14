import { InvitationProjectsLine } from "@/components/invitation/InvitationProjectsLine";
import type { InvitationGrant, InvitationRole } from "@/models/Invitation";

const InvitationRoleItem = ({ role }: { role: InvitationRole }) => (
  <li className="px-3.5 py-3">
    <p className="text-sm font-medium break-words">{role.name}</p>
    {role.description && <p className="mt-0.5 text-[13px] leading-5 text-muted-foreground">{role.description}</p>}
  </li>
);

interface InvitationGrantCardProps {
  grant: InvitationGrant;
  // After sign-in the card's foot names the projects the roles let in; the public preview never carries them.
  signedIn: boolean;
}

// One workspace the link admits to: its name, then a card of the roles given there.
export const InvitationGrantCard = ({ grant, signedIn }: InvitationGrantCardProps) => (
  <section className="space-y-2">
    <h2 className="flex min-w-0 items-center gap-2 text-[13px] font-medium">
      <span aria-hidden className="grid size-5 shrink-0 place-items-center rounded-[5px] bg-surface-2 font-mono text-[10px] font-semibold text-muted-foreground ring-1 ring-border">
        {grant.workspace_name.charAt(0).toUpperCase()}
      </span>
      <span className="min-w-0 break-words">{grant.workspace_name}</span>
    </h2>
    <div className="overflow-hidden rounded-lg border bg-card shadow-card">
      <ul className="divide-y divide-border">
        {grant.roles.map((role) => (
          <InvitationRoleItem key={role.id} role={role} />
        ))}
      </ul>
      {signedIn && <InvitationProjectsLine projects={grant.projects ?? []} />}
    </div>
  </section>
);
