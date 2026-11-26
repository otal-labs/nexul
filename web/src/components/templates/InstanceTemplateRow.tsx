import { ChevronRight } from "lucide-react";
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

// The Interview template is its questions; the play of the same name is told apart by its group.
const KIND_SUFFIX: Partial<Record<Template["kind"], string>> = { ticket_body: " body", interview: " questions" };

export const InstanceTemplateRow = ({ template, onLeave }: InstanceTemplateRowProps) => {
  const [open, setOpen] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const remove = useDeleteInstanceTemplate();
  const { open: confirm } = useConfirmationDialog();
  const label = `${template.name}${KIND_SUFFIX[template.kind] ?? ""}`;
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
    <li data-leaving={leaving || undefined} className={leavingRowClass}>
      <div className="flex min-h-11 items-center gap-1">
        <button
          type="button"
          aria-expanded={open}
          aria-label={editLabel}
          onClick={() => setOpen(!open)}
          className="-mx-2 flex min-w-0 flex-1 items-center gap-3 rounded-md px-2 py-1.5 text-left transition-[background-color] duration-150 ease-standard hover:bg-accent"
        >
          <span className="block min-w-0 flex-1 truncate text-sm font-medium first-letter:uppercase">{label}</span>
          <TemplateEditedLine template={template} />
          {!template.edited && <span className="text-xs text-muted-foreground">Default</span>}
          <ChevronRight
            aria-hidden
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-[rotate] duration-200 ease-out",
              open && "rotate-90",
            )}
          />
        </button>
        {template.added && (
          <Button type="button" variant="ghost" size="sm" aria-label={`Delete ${label}`} onClick={() => void onDelete()}>
            Delete
          </Button>
        )}
      </div>
      {open && <InstanceTemplateEditor key={`${template.body}:${template.updated_at ?? ""}`} template={template} />}
    </li>
  );
};
