import { useState, type FocusEvent } from "react";

import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { ColorSwatchButton } from "@/components/settings/ColorSwatchButton";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { TicketTypeTemplateForm } from "@/components/settings/TicketTypeTemplateForm";
import { useFetchProjectTicketCounts } from "@/hooks/ProjectHooks";
import { useFetchTemplates, useResetTemplate } from "@/hooks/TemplateHooks";
import { useDeleteTicketType, useRenameTicketType } from "@/hooks/TicketTypeHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  SaveTicketTypeTemplateFormSchema,
  type SaveTicketTypeTemplateFormData,
  type TicketType,
} from "@/models/TicketType";
import { findTemplate } from "@/models/Template";
import { countLabel } from "@/utils/CountUtility";

// Only a type named like one of the instance's (task, bug, feature) has an instance template to compare with.
const templateState = (body: string, instanceBody: string | undefined) => {
  if (instanceBody !== undefined) return body === instanceBody ? "Instance template" : "Edited template";
  return body ? "Own template" : "No template";
};

interface TicketTypeRowProps {
  type: TicketType;
  projectId: string;
}

export const TicketTypeRow = ({ type, projectId }: TicketTypeRowProps) => {
  const renameTicketType = useRenameTicketType();
  const deleteTicketType = useDeleteTicketType();
  const [editing, setEditing] = useState(false);
  const { open: openTemplate } = useFormDialog();
  const openClone = useCloneTemplateDialog();
  const { open: confirm } = useConfirmationDialog();
  const { data: counts } = useFetchProjectTicketCounts(projectId);
  const { data: templates } = useFetchTemplates();
  const resetTemplate = useResetTemplate();
  const instance = findTemplate(templates, "ticket_body", type.name);
  const used = counts?.tickets_by_type[type.id] ?? 0;

  const editTemplate = () =>
    openTemplate<SaveTicketTypeTemplateFormData>({
      title: `Template for ${type.name}`,
      description: `New ${type.name} tickets start with this body. Existing tickets keep theirs.`,
      schema: SaveTicketTypeTemplateFormSchema,
      okLabel: "Save template",
      form: <TicketTypeTemplateForm typeId={type.id} />,
      formOptions: { defaultValues: { body_template: type.body_template } },
    });

  const reset = async () => {
    if (!instance) return;
    const ok = await confirm({
      title: "Reset to the instance template?",
      message: `New ${type.name} tickets start from the instance's body again.`,
      confirmLabel: "Reset",
    });
    if (ok) resetTemplate.mutate({ kind: "ticket_body", key: instance.key, at: { scope: "project", project_id: projectId } });
  };

  // The server keeps a type any ticket still uses, so the confirm only has to name the template.
  const remove = async () => {
    const ok = await confirm({
      title: `Delete ${type.name}?`,
      message: "Its body template goes with it.",
      confirmLabel: "Delete type",
    });
    if (ok) deleteTicketType.mutate(type.id);
  };

  // Full-replacement PATCH — resend the untouched field so it isn't silently cleared.
  const commitRename = (event: FocusEvent<HTMLInputElement>) => {
    const name = event.target.value.trim();
    if (name && name !== type.name) {
      void renameTicketType.mutateAsync({ id: type.id, name, color: type.color });
    }
    setEditing(false);
  };

  const colorButton = (
    <ColorSwatchButton
      permission="projects:write"
      verb="recolor ticket types"
      projectId={projectId}
      label={`Color for ${type.name}`}
      value={type.color}
      onChange={(color) => void renameTicketType.mutateAsync({ id: type.id, name: type.name, color })}
    />
  );
  const edited = instance !== undefined && type.body_template !== instance.body;

  return (
    <>
      {editing && (
        <li className="flex min-h-11 items-center">
          <input
            className="flex-1 rounded-md border border-input px-2 py-1 text-sm"
            aria-label="Ticket type name"
            defaultValue={type.name}
            autoFocus
            onBlur={commitRename}
            onKeyDown={(event) => {
              if (event.key === "Enter") (event.target as HTMLInputElement).blur();
              if (event.key === "Escape") setEditing(false);
            }}
          />
        </li>
      )}
      {!editing && (
        <SettingsListRow
          lead={colorButton}
          name={<span className="truncate font-medium" title={type.name}>{type.name}</span>}
          meta={[!HUE_DOT_CLASS[type.color] && "No color", templates && templateState(type.body_template, instance?.body)].filter(Boolean).join(" · ") || undefined}
          count={counts && countLabel(used, "ticket")}
          actions={
            <RowActionsMenu
              subject={type.name}
              projectId={projectId}
              actions={[
                { label: "Rename", permission: "projects:write", verb: "rename ticket types", onSelect: () => setEditing(true) },
                { label: "Edit template", permission: "projects:write", verb: "edit ticket type templates", onSelect: () => void editTemplate() },
                ...(edited ? [{ label: "Reset to the instance template", permission: "templates:write" as const, verb: "reset templates", onSelect: () => void reset() }] : []),
                {
                  label: "Clone template to…",
                  permission: "templates:write",
                  onSelect: () =>
                    void openClone({ kind: "ticket_body", key: type.name, name: type.name, from: { scope: "project", project_id: projectId } }),
                },
                {
                  label: "Delete",
                  destructive: true,
                  permission: "projects:delete",
                  verb: "delete ticket types",
                  disabled: used > 0,
                  hint: used > 0 ? `Used by ${countLabel(used, "ticket")}` : undefined,
                  onSelect: () => void remove(),
                },
              ]}
            />
          }
        />
      )}
    </>
  );
};
