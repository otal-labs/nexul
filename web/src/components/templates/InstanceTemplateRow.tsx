import { useState } from "react";

import { InstanceTemplateEditor } from "@/components/templates/InstanceTemplateEditor";
import { TemplateEditedLine } from "@/components/templates/TemplateEditedLine";
import { Button } from "@/components/ui/button";
import type { Template } from "@/models/Template";

interface InstanceTemplateRowProps {
  template: Template;
}

export const InstanceTemplateRow = ({ template }: InstanceTemplateRowProps) => {
  const [open, setOpen] = useState(false);
  const label = `${template.name}${template.kind === "ticket_body" ? " body" : ""}`;
  // The Interview play shares its name with the Interview template, so its button names the instructions.
  const editLabel = `Edit ${label}${template.kind === "play_instructions" ? " instructions" : ""}`;

  return (
    <li className="py-3">
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium first-letter:uppercase">{label}</p>
          <TemplateEditedLine template={template} />
        </div>
        <Button type="button" variant="ghost" size="sm" aria-expanded={open} aria-label={editLabel} onClick={() => setOpen(!open)}>
          {open ? "Close" : "Edit"}
        </Button>
      </div>
      {open && <InstanceTemplateEditor key={`${template.body}:${template.updated_at ?? ""}`} template={template} />}
    </li>
  );
};
