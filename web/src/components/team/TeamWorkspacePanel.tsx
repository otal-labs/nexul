import { ArrowUpRight } from "lucide-react";
import { Link } from "react-router";

import { SettingsHint } from "@/components/settings/SettingsHint";
import { Microheader } from "@/components/Microheader";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { TeamViewAsButton } from "@/components/team/TeamViewAsButton";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { heldRoles, type TeamMembership, type TeamPerson, type TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamWorkspacePanelProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
  membership: TeamMembership;
}

// Read-only: a person's roles are given and taken from each role on the workspace's Access page (ADR 0198).
export const TeamWorkspacePanel = ({ person, workspace, membership }: TeamWorkspacePanelProps) => {
  const { data: workspaces } = useFetchWorkspaces();
  const slug = workspaces?.find((candidate) => candidate.id === workspace.id)?.slug;
  const firstName = personName(person).split(" ")[0] ?? "";

  return (
    <div className="space-y-5">
      <section className="space-y-2">
        <Microheader>Roles</Microheader>
        <RoleChipRow roles={heldRoles(membership)} />
        {membership.is_owner && <SettingsHint>The Owner role is never given or taken here, and it holds everything in {workspace.name}.</SettingsHint>}
        {!membership.is_owner && (
          <SettingsHint>
            The projects, channels and plays {firstName} sees follow from these roles and from the rules on each one.
          </SettingsHint>
        )}
        {!membership.is_owner && workspace.can_manage_members && slug && (
          <Link to={`/${slug}/access/roles`} className="inline-flex items-center gap-1 text-sm font-medium underline-offset-4 hover:underline">
            Change on Access
            <ArrowUpRight aria-hidden className="size-3.5" />
          </Link>
        )}
      </section>
      {!membership.is_owner && <TeamViewAsButton person={person} workspace={workspace} />}
    </div>
  );
};
