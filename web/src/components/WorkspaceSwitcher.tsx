import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { CreateWorkspaceForm } from "@/components/CreateWorkspaceForm";
import { SwitcherTrigger } from "@/components/SwitcherTrigger";
import { Popover } from "@/components/ui/popover";
import { WorkspaceSwitcherMenu } from "@/components/WorkspaceSwitcherMenu";
import { useHasInstancePermission, workspaceAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { myRoleQuery, useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { workspaceWidePermissions } from "@/models/Permission";
import { SaveWorkspaceFormSchema, switchWorkspacePath, workspacePath, type SaveWorkspaceFormData } from "@/models/Workspace";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface WorkspaceSwitcherProps {
  collapsed: boolean;
}

// Sits above the sidebar nav, per the workspace-sidebar-redesign spec (.scratch/workspace-sidebar-redesign/spec.md).
export const WorkspaceSwitcher = ({ collapsed }: WorkspaceSwitcherProps) => {
  const [open, setOpen] = useState(false);
  // Server state, rendered straight from the query cache, never mirrored into workspaceStore (F5).
  const { data: workspaces } = useFetchWorkspaces();
  const canCreateWorkspace = useHasInstancePermission("workspaces:create");
  const { selectedWorkspaceId, selectWorkspace } = useWorkspaceStore(
    useShallow((s) => ({
      selectedWorkspaceId: s.selectedWorkspaceId,
      selectWorkspace: s.selectWorkspace,
    })),
  );

  const current = workspaces?.find((w) => w.id === selectedWorkspaceId) ?? workspaces?.[0];
  const { open: openCreate } = useFormDialog();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const client = useQueryClient();
  const { data: me } = useFetchMe();

  const handleCreate = async () => {
    setOpen(false);
    const result = await openCreate<SaveWorkspaceFormData>({
      title: "New workspace",
      schema: SaveWorkspaceFormSchema,
      okLabel: "Create workspace",
      form: <CreateWorkspaceForm />,
      formOptions: { defaultValues: { name: "" } },
    });
    // The dialog renders outside the router, so the move into the new workspace happens here.
    const created = result.data as (SaveWorkspaceFormData & { slug?: string }) | null;
    if (created?.slug) void navigate(workspacePath(created.slug, "/"));
  };

  // Inside a workspace the URL decides: the same section in the other one. A personal page has no workspace URL to change.
  const handleSelect = async (workspaceId: string) => {
    setOpen(false);
    const target = workspaces?.find((w) => w.id === workspaceId);
    const prefix = current ? workspacePath(current.slug, "/") : "";
    const inWorkspace = !!current && (pathname === prefix || pathname.startsWith(`${prefix}/`));
    if (!target || !inWorkspace) return selectWorkspace(workspaceId, target?.slug ?? "");
    const role = await client.fetchQuery(myRoleQuery(workspaceId)).catch(() => undefined);
    void navigate(switchWorkspacePath(pathname, target.slug, workspaceAccess(me?.instance_permissions, workspaceWidePermissions(role))));
  };

  // Nothing to switch between yet; same "render nothing" convention as AccountMenu.
  if (!current) {
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <SwitcherTrigger tile={current.name[0] ?? ""} name={current.name} collapsed={collapsed} />
      <WorkspaceSwitcherMenu
        workspaces={workspaces}
        selectedWorkspaceId={selectedWorkspaceId}
        onSelect={(id) => void handleSelect(id)}
        canCreateWorkspace={canCreateWorkspace}
        onCreate={handleCreate}
      />
    </Popover>
  );
};
