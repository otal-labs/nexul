import { Eye } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useStartViewAs } from "@/hooks/ViewAsHooks";
import type { TeamPerson, TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamViewAsButtonProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
}

// Checks what a member's roles give them in one workspace; you never view as yourself, and an Owner sees everything.
export const TeamViewAsButton = ({ person, workspace }: TeamViewAsButtonProps) => {
  const { data: me } = useFetchMe();
  const start = useStartViewAs();
  const firstName = personName(person).split(" ")[0] ?? "";

  if (!me || me.user.id === person.id) return null;

  return (
    <ActionButton
      permission="members:view_as"
      type="button"
      variant="outline"
      size="sm"
      icon={Eye}
      loading={start.isPending}
      onClick={() => start.mutate({ workspaceId: workspace.id, workspaceName: workspace.name, userId: person.id })}
    >
      View as {firstName}
    </ActionButton>
  );
};
