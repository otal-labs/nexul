import { useState, type FocusEvent } from "react";

import { ColorSwatchButton } from "@/components/settings/ColorSwatchButton";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { TicketTypeTemplateForm } from "@/components/settings/TicketTypeTemplateForm";
import { TicketTypeTemplateLine } from "@/components/settings/TicketTypeTemplateLine";
import { useDeleteTicketType, useRenameTicketType } from "@/hooks/TicketTypeHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  SaveTicketTypeTemplateFormSchema,
  type SaveTicketTypeTemplateFormData,
  type TicketType,
} from "@/models/TicketType";

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

  const editTemplate = () =>
    openTemplate<SaveTicketTypeTemplateFormData>({
      title: `Template for ${type.name}`,
      description: `New ${type.name} tickets start with this body. Existing tickets keep theirs.`,
      schema: SaveTicketTypeTemplateFormSchema,
      okLabel: "Save template",
      form: <TicketTypeTemplateForm typeId={type.id} />,
      formOptions: { defaultValues: { body_template: type.body_template } },
    });

  // Full-replacement PATCH — resend the untouched field so it isn't silently cleared.
  const commitRename = (event: FocusEvent<HTMLInputElement>) => {
    const name = event.target.value.trim();
    if (name && name !== type.name) {
      void renameTicketType.mutateAsync({ id: type.id, name, color: type.color });
    }
    setEditing(false);
  };

  return (
    <li className="-mx-2 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      {editing && (
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
      )}
      {!editing && (
        <div className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium" title={type.name}>{type.name}</span>
          <TicketTypeTemplateLine type={type} projectId={projectId} />
        </div>
      )}
      <ColorSwatchButton
        label={`Color for ${type.name}`}
        value={type.color}
        onChange={(color) => void renameTicketType.mutateAsync({ id: type.id, name: type.name, color })}
      />
      <RowActionsMenu
        subject={type.name}
        actions={[
          { label: "Rename", onSelect: () => setEditing(true) },
          { label: "Edit template", onSelect: () => void editTemplate() },
          {
            label: "Clone template to…",
            onSelect: () =>
              void openClone({ kind: "ticket_body", key: type.name, name: type.name, from: { scope: "project", project_id: projectId } }),
          },
          { label: "Delete", destructive: true, onSelect: () => deleteTicketType.mutate(type.id) },
        ]}
      />
    </li>
  );
};
