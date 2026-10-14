import { useMemo, useState } from "react";
import { WELCOME_STEP_NAMES } from "@nexul/client-core/welcome";

import { RoleAccessSummary } from "@/components/settings/RoleAccessSummary";
import { RoleEditorDialog } from "@/components/settings/RoleEditorDialog";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { TemplateEditedLine } from "@/components/templates/TemplateEditedLine";
import { useDeleteInstanceTemplate, useResetInstanceTemplate } from "@/hooks/TemplateHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import type { PermissionInfo } from "@/models/Permission";
import { ROLE_TEMPLATE, type Template } from "@/models/Template";
import { templateRole } from "@/models/TemplateRole";
import { cn } from "@/lib/utils";

interface TemplateRoleRowProps {
  template: Template;
  catalog: PermissionInfo[];
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave: () => void;
}

const NO_START: never[] = [];

// One template role: what it allows and asks, with Reset for the shipped ones and Delete for the instance's own.
export const TemplateRoleRow = ({ template, catalog, onLeave }: TemplateRoleRowProps) => {
  const reset = useResetInstanceTemplate();
  const remove = useDeleteInstanceTemplate();
  const { open: confirm } = useConfirmationDialog();
  const [editing, setEditing] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const role = useMemo(() => templateRole(template), [template]);
  const steps = role.welcome_steps.map((step) => WELCOME_STEP_NAMES[step]).join(", ");

  const onReset = async () => {
    const ok = await confirm({
      title: `Reset ${role.name}?`,
      message: "It goes back to the permissions and welcome steps Nexul ships with. Workspaces that have a copy keep theirs.",
      confirmLabel: "Reset",
    });
    if (ok) reset.mutate({ kind: ROLE_TEMPLATE, key: template.key });
  };

  const onDelete = async () => {
    const ok = await confirm({
      title: `Delete ${role.name}?`,
      message: "New workspaces stop getting it. Workspaces that have a copy keep theirs.",
      confirmLabel: "Delete template role",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave();
    remove.mutate({ kind: ROLE_TEMPLATE, key: template.key }, { onError: () => setLeaving(false) });
  };

  const actions: RowAction[] = [
    { label: "Edit", onSelect: () => setEditing(true) },
    ...(template.added ? [] : [{ label: "Reset to default", disabled: !template.edited, onSelect: () => void onReset() }]),
    ...(template.added ? [{ label: "Delete", destructive: true, onSelect: () => void onDelete() }] : []),
  ];

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "space-y-2.5 bg-card px-4 py-3.5")}>
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium" title={role.name}>
            {role.name}
          </p>
          <TemplateEditedLine template={template} />
        </div>
        <RowActionsMenu subject={role.name} actions={actions} />
      </div>
      <RoleAccessSummary catalog={catalog} permissions={role.permissions} />
      <p className="truncate text-xs text-muted-foreground" title={steps}>
        <span className="text-foreground/80">Welcome</span> · {steps || "No steps"}
      </p>
      <RoleEditorDialog open={editing} role={role} template={{ key: template.key, startFrom: NO_START }} onClose={() => setEditing(false)} />
    </li>
  );
};
