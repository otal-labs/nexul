import { SwitcherMenu, SwitcherMenuItem } from "@/components/SwitcherMenu";
import { useFetchUnreadByWorkspace } from "@/hooks/NotificationHooks";
import type { Workspace } from "@/models/Workspace";

interface WorkspaceSwitcherMenuProps {
  workspaces: Workspace[] | undefined;
  selectedWorkspaceId: string;
  onSelect: (workspaceId: string) => void;
  canCreateWorkspace: boolean;
  onCreate: () => void;
  collapsed: boolean;
}

export const WorkspaceSwitcherMenu = ({
  workspaces,
  selectedWorkspaceId,
  onSelect,
  canCreateWorkspace,
  onCreate,
  collapsed,
}: WorkspaceSwitcherMenuProps) => {
  const { data: unread } = useFetchUnreadByWorkspace();
  return (
    <SwitcherMenu createLabel="New workspace" onCreate={canCreateWorkspace ? onCreate : undefined} collapsed={collapsed}>
      {workspaces?.map((ws) => (
        <SwitcherMenuItem
          key={ws.id}
          tile={ws.name[0] ?? ""}
          name={ws.name}
          selected={ws.id === selectedWorkspaceId}
          onSelect={() => onSelect(ws.id)}
          unreadCount={unread?.[ws.id] ?? 0}
        />
      ))}
    </SwitcherMenu>
  );
};
