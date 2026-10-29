import { ShieldCheck } from "lucide-react";

import { InvitationProjectsLine } from "@/components/invitation/InvitationProjectsLine";
import type { InvitationGrant, InvitationRole } from "@/models/Invitation";

// The role mark keeps a lone undescribed role reading as a list row, not an empty field.
const InvitationRoleItem = ({ role }: { role: InvitationRole }) => (
  <li className="flex gap-2.5 px-3.5 py-1.5">
    <ShieldCheck aria-hidden className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
    <div className="min-w-0">
      <p className="text-sm leading-5 font-medium break-words text-pretty">{role.name}</p>
      {role.description && <p className="mt-0.5 text-[13px] leading-[18px] text-pretty text-muted-foreground">{role.description}</p>}
    </div>
  </li>
);

interface InvitationGrantGroupProps {
  grant: InvitationGrant;
  // After sign-in the group counts and names the projects the roles let in; the public preview never carries them.
  signedIn: boolean;
}

const projectCount = (count: number): string => {
  if (count === 0) return "No projects yet";
  return `${count} ${count === 1 ? "project" : "projects"}`;
};

// One workspace the link admits to: its name, then a card of the roles given there with what each allows.
export const InvitationGrantGroup = ({ grant, signedIn }: InvitationGrantGroupProps) => {
  const projects = grant.projects ?? [];
  return (
    <section className="space-y-1.5">
      <div className="flex min-w-0 items-baseline gap-2">
        <span aria-hidden className="mt-px grid self-start size-[18px] shrink-0 place-items-center rounded-sm bg-surface-2 font-mono text-[10px] font-semibold text-muted-foreground ring-1 ring-border">
          {grant.workspace_name.charAt(0).toUpperCase()}
        </span>
        <h2 title={grant.workspace_name} className="min-w-0 flex-1 text-sm leading-5 font-medium break-words">{grant.workspace_name}</h2>
        {signedIn && <span className="shrink-0 text-xs leading-5 text-muted-foreground tabular-nums">{projectCount(projects.length)}</span>}
      </div>
      <div className="overflow-hidden rounded-lg border bg-card shadow-card">
        <ul aria-label={`Roles in ${grant.workspace_name}`} className="divide-y divide-border">
          {grant.roles.map((role) => <InvitationRoleItem key={role.id} role={role} />)}
        </ul>
        {signedIn && projects.length > 0 && <InvitationProjectsLine projects={projects} />}
      </div>
    </section>
  );
};
