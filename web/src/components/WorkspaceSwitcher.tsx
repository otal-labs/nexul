import { useState } from "react";
import { useNavigate } from "react-router";

import { CreateWorkspaceForm } from "@/components/CreateWorkspaceForm";
import { SwitcherTrigger } from "@/components/SwitcherTrigger";
import { Popover } from "@/components/ui/popover";
import { WorkspaceSwitcherMenu } from "@/components/WorkspaceSwitcherMenu";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useSwitchWorkspace } from "@/hooks/useSwitchWorkspace";
import { SaveWorkspaceFormSchema, workspacePath, type SaveWorkspaceFormData } from "@/models/Workspace";

interface WorkspaceSwitcherProps {
  collapsed: boolean;
}

export const WorkspaceSwitcher = ({ collapsed }: WorkspaceSwitcherProps) => {
  const [open, setOpen] = useState(false);
  const { workspaces, current, selectedWorkspaceId, switchTo } = useSwitchWorkspace();
  const canCreateWorkspace = useHasInstancePermission("workspaces:create");
  const { open: openCreate } = useFormDialog();
  const navigate = useNavigate();

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

  const handleSelect = (workspaceId: string) => {
    setOpen(false);
    void switchTo(workspaceId);
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
