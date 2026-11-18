import { PersonAvatar } from "@/components/PersonAvatar";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { RoleGivePopover } from "@/components/role/RoleGivePopover";
import { rowClass } from "@/components/role/rowStyles";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
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
  // "Signed out" on every row of a long list says nothing; presence shows once it does.
  const seen = person.online || person.last_seen_at !== null;
  const meta = seen || person.status !== "active" || others.length > 0;

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
    <li className="flex min-h-14 border-b border-border py-1">
      <div className={rowClass}>
        <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-7" />
        <span className="min-w-0 flex-1">
          <span dir="auto" className="block truncate text-sm font-medium" title={personName(person)}>
            {personName(person)}
          </span>
          {meta && (
            <span className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
              {seen && <SettingsStatus tone={person.online ? "success" : "muted"}>{presenceText(person)}</SettingsStatus>}
              <AccountStatusLabel status={person.status} />
              {others.length > 0 && (
                <span className="-my-0.5 flex min-w-0 flex-1">
                  <RoleChipRow roles={others} />
                </span>
              )}
            </span>
          )}
        </span>
        {!owner && !role && <RoleGivePopover person={person} />}
        {!owner && <RowActionsMenu subject={personName(person)} actions={actions} />}
      </div>
    </li>
  );
};
