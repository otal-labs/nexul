import { useState } from "react";
import { useShallow } from "zustand/react/shallow";

import { CreateWorkspaceForm } from "@/components/CreateWorkspaceForm";
import { SwitcherTrigger } from "@/components/SwitcherTrigger";
import { Popover } from "@/components/ui/popover";
import { WorkspaceSwitcherMenu } from "@/components/WorkspaceSwitcherMenu";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveWorkspaceFormSchema, type SaveWorkspaceFormData } from "@/models/Workspace";
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

  const handleCreate = async () => {
    setOpen(false);
    await openCreate<SaveWorkspaceFormData>({
      title: "New workspace",
      schema: SaveWorkspaceFormSchema,
      okLabel: "Create workspace",
      form: <CreateWorkspaceForm />,
      formOptions: { defaultValues: { name: "" } },
    });
  };

  const handleSelect = (workspaceId: string) => {
    selectWorkspace(workspaceId);
    setOpen(false);
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
        onSelect={handleSelect}
        canCreateWorkspace={canCreateWorkspace}
        onCreate={handleCreate}
      />
    </Popover>
  );
};
