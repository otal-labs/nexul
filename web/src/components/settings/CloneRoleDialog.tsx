import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCloneWorkspaceRole } from "@/hooks/RoleHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { Role } from "@/models/Role";

const cloneRoleSchema = z.object({
  workspaceId: z.string().trim().min(1, "Pick a workspace"),
});

type CloneRoleFormData = z.infer<typeof cloneRoleSchema>;

interface CloneRoleDialogProps {
  role: Role;
  open: boolean;
  onClose: () => void;
}

export const CloneRoleDialog = ({ role, open, onClose }: CloneRoleDialogProps) => {
  const { data: workspaces, error, isPending } = useFetchWorkspaces();
  const clone = useCloneWorkspaceRole();
  const form = useForm<CloneRoleFormData>({
    resolver: zodResolver(cloneRoleSchema),
    defaultValues: { workspaceId: "" },
  });

  const targets = workspaces?.filter((workspace) => workspace.id !== role.workspace_id);
  const hasTargets = !!targets && targets.length > 0;
  const options = (targets ?? []).map((workspace) => ({ value: workspace.id, label: workspace.name }));

  const onSubmit = async (data: CloneRoleFormData) => {
    const target = targets?.find((workspace) => workspace.id === data.workspaceId);
    if (!target) return;
    try {
      await clone.mutateAsync({ role, target });
      onClose();
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Clone {role.name} to another workspace</DialogTitle>
          <DialogDescription>
            The copy keeps this role's name and permissions. If the workspace already has a role with that name, the
            copy is named "{role.name} (copy)".
          </DialogDescription>
        </DialogHeader>
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {targets && targets.length === 0 && (
          <EmptyRow>You aren't in any other workspace to clone this role into.</EmptyRow>
        )}
        {hasTargets && (
          <form id="clone-role-form" onSubmit={form.handleSubmit(onSubmit)}>
            <FormSelect
              control={form.control}
              name="workspaceId"
              label="Workspace"
              options={options}
              placeholder="Select a workspace"
            />
          </form>
        )}
        <DialogFooter>
          <Button variant="outline" type="button" onClick={onClose}>
            Cancel
          </Button>
          {hasTargets && (
            <Button type="submit" form="clone-role-form" loading={clone.isPending}>
              Clone
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
