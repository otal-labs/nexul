import { TeamMembershipItem } from "@/components/team/TeamMembershipItem";
import { TeamNonMemberItem } from "@/components/team/TeamNonMemberItem";
import type { TeamPerson, TeamWorkspace } from "@/models/Team";
import { membershipIn } from "@/utils/TeamUtility";

interface TeamWorkspaceAccessRowProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
}

export const TeamWorkspaceAccessRow = ({ person, workspace }: TeamWorkspaceAccessRowProps) => {
  const membership = membershipIn(person, workspace.id);
  return (
    <li aria-label={workspace.name} className="space-y-2 px-3 py-3 sm:px-4">
      {membership && <TeamMembershipItem person={person} workspace={workspace} membership={membership} />}
      {!membership && <TeamNonMemberItem person={person} workspace={workspace} />}
    </li>
  );
};
