import { useState } from "react";
import type { WelcomeStep } from "@nexul/client-core/welcome";

import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { RoleDialogFooter } from "@/components/settings/RoleDialogFooter";
import { RoleHolderLine } from "@/components/settings/RoleHolderLine";
import { RoleNameFields } from "@/components/settings/RoleNameFields";
import { RoleSectionTabs } from "@/components/settings/RoleSectionTabs";
import { useCreateWorkspaceRole, useUpdateWorkspaceRole } from "@/hooks/RoleHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { RoleDraftContext } from "@/hooks/useRoleDraft";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { PermissionArea } from "@/models/Permission";
import type { Role } from "@/models/Role";
import { NO_ACTIONS, NO_STEPS, sameActions, type RoleTab } from "@/models/RoleSection";
import type { TeamPerson } from "@/models/Team";

interface RoleEditorDialogProps {
  open: boolean;
  // The role to edit; none opens it as New role.
  role?: Role;
  holders?: TeamPerson[] | undefined;
  onClose: () => void;
}

interface RoleEdits {
  name?: string;
  actions?: string[];
  steps?: WelcomeStep[];
  copiedFrom?: Role | undefined;
}

// New role and Edit in one dialog: changes are held over what was loaded until Save, and leaving with them asks first.
export const RoleEditorDialog = ({ open, role, holders, onClose }: RoleEditorDialogProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const createRole = useCreateWorkspaceRole(workspaceId);
  const updateRole = useUpdateWorkspaceRole(workspaceId);
  const { open: ask } = useConfirmationDialog();
  const [edits, setEdits] = useState<RoleEdits>({});
  const [tab, setTab] = useState<RoleTab>(PermissionArea.Instance);

  const creating = !role;
  const loadedName = role?.name ?? "";
  const loaded = role?.permissions ?? NO_ACTIONS;
  const name = edits.name ?? loadedName;
  const actions = edits.actions ?? loaded;
  const loadedSteps = role?.welcome_steps ?? NO_STEPS;
  const steps = edits.steps ?? loadedSteps;
  const named = name.trim().length > 0;
  const dirty = name !== loadedName || !sameActions(actions, loaded) || !sameActions(steps, loadedSteps);
  const title = creating ? "New role" : `Edit ${role.name}`;

  const finish = () => {
    setEdits({});
    setTab(PermissionArea.Instance);
    onClose();
  };

  const leave = async () => {
    if (!dirty) {
      finish();
      return;
    }
    const discard = await ask({
      title: "Discard changes?",
      message: creating ? "This role hasn't been created yet; what you set here will be lost." : `Your changes to ${role.name} haven't been saved.`,
      confirmLabel: "Discard changes",
    });
    if (discard) finish();
  };

  const save = () => {
    const input = { name: name.trim(), actions, welcome_steps: steps };
    if (!role) {
      createRole.mutate(input, { onSuccess: finish });
      return;
    }
    updateRole.mutate({ roleId: role.id, ...input }, { onSuccess: finish });
  };

  const draft = {
    actions,
    loaded,
    setActions: (next: string[]) => setEdits((current) => ({ ...current, actions: next })),
    steps,
    loadedSteps,
    setSteps: (next: WelcomeStep[]) => setEdits((current) => ({ ...current, steps: next })),
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && void leave()}>
      <DialogContent
        onOpenAutoFocus={(event) => {
          // Lands on the dialog itself: no field takes focus, and the row menu's tooltip doesn't linger above the scrim.
          event.preventDefault();
          (event.currentTarget as HTMLElement).focus();
        }}
        className="h-[min(52rem,calc(100dvh-4rem))] gap-0 overflow-hidden p-0 sm:max-w-[40rem]"
      >
        <DialogHeader className="gap-1 px-6 pt-6 pb-4 pr-12">
          <DialogTitle dir="auto" className="line-clamp-2 break-words" title={title}>
            {title}
          </DialogTitle>
          {creating && <DialogDescription>Nobody holds a new role until you give it to someone in Team.</DialogDescription>}
          {!creating && <RoleHolderLine holders={holders ?? []} />}
        </DialogHeader>
        <div className="border-t border-border px-6 pt-5 pb-4">
          <RoleNameFields
            creating={creating}
            name={name}
            copiedFrom={edits.copiedFrom}
            onName={(next) => setEdits((current) => ({ ...current, name: next }))}
            onStartFrom={(from) =>
              setEdits((current) => ({
                ...current,
                copiedFrom: from,
                actions: from?.permissions ?? NO_ACTIONS,
                steps: from?.welcome_steps ?? NO_STEPS,
              }))
            }
          />
        </div>
        <RoleDraftContext value={draft}>
          <RoleSectionTabs tab={tab} onTab={setTab} />
        </RoleDraftContext>
        <RoleDialogFooter
          creating={creating}
          dirty={dirty}
          named={named}
          saving={createRole.isPending || updateRole.isPending}
          onCancel={() => void leave()}
          onSave={save}
        />
      </DialogContent>
    </Dialog>
  );
};
