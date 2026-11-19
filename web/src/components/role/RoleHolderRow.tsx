import { MatchText } from "@/components/MatchText";
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
import { cn } from "@/lib/utils";
import { heldRoles, type TeamPerson } from "@/models/Team";
import { loginHandle } from "@/models/User";
import { personName, presenceText } from "@/utils/TeamUtility";

interface RoleHolderRowProps {
  person: TeamPerson;
  // Opens the person's account dialog, the one Settings → Team opens.
  onOpen: (personId: string) => void;
  // The search narrowing the list, marked in the name and handle.
  match?: string | undefined;
}

// A person in the open role (or in No roles): the whole row opens them, with their presence and other roles, and a menu to take
// the role, view as them, or take them out of the workspace. An Owner's row has no menu; the Owner role is never given or taken here.
export const RoleHolderRow = ({ person, onOpen, match }: RoleHolderRowProps) => {
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
    <li className="group/row flex min-h-14 border-b border-border py-1">
      <div className={rowClass}>
        <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className={cn("size-7", person.status !== "active" && "opacity-50 grayscale")} />
        <button
          type="button"
          onClick={() => onOpen(person.id)}
          aria-label={`Open ${personName(person)}`}
          className="quiet-focus min-w-0 flex-1 text-left after:absolute after:inset-0 after:rounded-md focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-focus"
        >
          <span dir="auto" className={cn("block truncate text-sm font-medium", person.status !== "active" && "text-muted-foreground")} title={personName(person)}>
            <MatchText text={personName(person)} match={match} />
          </span>
          {/* The handle leads, so two people with like names tell apart; presence, status and other roles follow. */}
          <span className="flex min-w-0 items-center gap-4 text-xs text-muted-foreground">
            <span className="shrink-0 font-mono">
              <MatchText text={loginHandle(person.login)} match={match} />
            </span>
            {person.online && <SettingsStatus tone="success">Online</SettingsStatus>}
            {seen && !person.online && <span className="shrink-0">{presenceText(person)}</span>}
            <AccountStatusLabel status={person.status} />
            {others.length > 0 && (
              <span className="-my-0.5 flex min-w-0 flex-1">
                <RoleChipRow roles={others} />
              </span>
            )}
          </span>
        </button>
        {/* The menu comes first, so on No roles the one action, Give a role, holds the row's edge while the menu waits for hover. */}
        {!owner && (
          <span className="relative [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:has-data-[state=open]:opacity-100">
            <RowActionsMenu subject={personName(person)} actions={actions} />
          </span>
        )}
        {!owner && !role && (
          <span className="relative">
            <RoleGivePopover person={person} />
          </span>
        )}
      </div>
    </li>
  );
};
