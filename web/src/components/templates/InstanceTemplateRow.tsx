import { useState } from "react";

import { InstanceTemplateEditor } from "@/components/templates/InstanceTemplateEditor";
import { TemplateEditedLine } from "@/components/templates/TemplateEditedLine";
import { Button } from "@/components/ui/button";
import { useDeleteInstanceTemplate } from "@/hooks/TemplateHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { leavingRowClass } from "@/hooks/useRowGlide";
import type { Template } from "@/models/Template";
import { cn } from "@/lib/utils";

interface InstanceTemplateRowProps {
  template: Template;
  // Called as an added template's row starts to leave, so the group can glide the rows under it up once it is gone.
  onLeave: () => void;
}

export const InstanceTemplateRow = ({ template, onLeave }: InstanceTemplateRowProps) => {
  const [open, setOpen] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const remove = useDeleteInstanceTemplate();
  const { open: confirm } = useConfirmationDialog();
  const label = `${template.name}${template.kind === "ticket_body" ? " body" : ""}`;
  // The Interview play shares its name with the Interview template, so its button names the instructions.
  const editLabel = `Edit ${label}${template.kind === "play_instructions" ? " instructions" : ""}`;

  const onDelete = async () => {
    const ok = await confirm({
      title: `Delete ${label}?`,
      message: "New projects stop getting it. Projects that have a copy keep theirs.",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    setLeaving(true);
    onLeave();
    remove.mutate({ kind: template.kind, key: template.key }, { onError: () => setLeaving(false) });
  };

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "group/row py-2")}>
      <div className="flex min-h-9 items-center gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium first-letter:uppercase">{label}</p>
          <TemplateEditedLine template={template} />
        </div>
        {template.added && (
          <Button type="button" variant="ghost" size="sm" aria-label={`Delete ${label}`} onClick={() => void onDelete()}>
            Delete
          </Button>
        )}
        {/* Quiet until the row is pointed at, where something can hover; an open editor keeps its Close. */}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-expanded={open}
          aria-label={editLabel}
          onClick={() => setOpen(!open)}
          className={cn(
            "-mr-3",
            !open &&
              "[@media(hover:hover)]:opacity-0 [@media(hover:hover)]:transition-opacity [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:focus-visible:opacity-100",
          )}
        >
          {open ? "Close" : "Edit"}
        </Button>
      </div>
      {open && <InstanceTemplateEditor key={`${template.body}:${template.updated_at ?? ""}`} template={template} />}
    </li>
  );
};
