import { PersonAvatar } from "@/components/PersonAvatar";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useOpenRole } from "@/hooks/RoleHooks";
import { useRoleHolders } from "@/hooks/TeamHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useStartViewAs } from "@/hooks/ViewAsHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { heldRoles, type TeamPerson } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";

interface RoleHolderRowProps {
  person: TeamPerson;
}

// A person in the open role (or in No roles): their presence and other roles, and a menu to take the role, view as
// them, or take them out of the workspace. An Owner's row has no menu; the Owner role is never given or taken here.
export const RoleHolderRow = ({ person }: RoleHolderRowProps) => {
  const workspace = useSelectedWorkspace();
  const workspaceName = workspace?.name ?? "this workspace";
  const { role, givable, reason } = useOpenRole();
  const holders = useRoleHolders(workspace?.id ?? "", workspaceName);
  const viewAs = useStartViewAs();
  const { data: me } = useFetchMe();
  const { open: confirm } = useConfirmationDialog();
  const membership = person.workspaces.find((candidate) => candidate.workspace_id === workspace?.id);
  const others = membership ? heldRoles(membership).filter((held) => held.id && held.id !== role?.id) : [];
  const firstName = personName(person).split(" ")[0] ?? "";
  const owner = membership?.is_owner === true;

  const remove = async () => {
    const ok = await confirm({
      title: `Remove ${personName(person)} from ${workspaceName}?`,
      message: `They lose every role here and stop seeing its projects, channels and plays. Their account stays.`,
      confirmLabel: "Remove",
    });
    if (ok) holders.remove(person);
  };

  const actions: RowAction[] = [
    ...(role && !role.is_owner_role
      ? [{ label: `Take ${role.name}`, permission: "members:write" as const, verb: "take roles", disabled: !givable, hint: givable ? undefined : reason, onSelect: () => holders.take(person, role) }]
      : []),
    ...(me && me.user.id !== person.id && workspace
      ? [{ label: `View as ${firstName}`, permission: "members:view_as" as const, onSelect: () => viewAs.mutate({ workspaceId: workspace.id, workspaceName: workspace.name, userId: person.id }) }]
      : []),
    { label: `Remove from ${workspaceName}`, destructive: true, permission: "members:write", verb: "remove members", onSelect: () => void remove() },
  ];

  return (
    <li className="flex min-h-14 items-center gap-3 border-b border-border py-2.5">
      <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-7" />
      <span className="min-w-0 flex-1">
        <span dir="auto" className="block truncate text-sm font-medium" title={personName(person)}>
          {personName(person)}
        </span>
        <span className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
          <span className="shrink-0">{presenceText(person)}</span>
          {others.length > 0 && (
            <span className="flex min-w-0 overflow-x-clip">
              <RoleChipRow roles={others} />
            </span>
          )}
        </span>
      </span>
      {!owner && <RowActionsMenu subject={personName(person)} actions={actions} />}
    </li>
  );
};
