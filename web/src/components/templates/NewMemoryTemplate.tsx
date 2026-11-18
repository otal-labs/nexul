import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { InstanceTemplateEditor } from "@/components/templates/InstanceTemplateEditor";
import { Button } from "@/components/ui/button";
import { MEMORY_TEMPLATE, MEMORY_TEMPLATE_SKELETON, type Template } from "@/models/Template";

// A blank key adds a template under a generated one; added is true, so the editor offers no reset.
const draft: Template = {
  kind: MEMORY_TEMPLATE,
  key: "",
  name: "New memory",
  scope: "instance",
  body: MEMORY_TEMPLATE_SKELETON,
  default_body: MEMORY_TEMPLATE_SKELETON,
  edited: false,
  follows: false,
  added: true,
};

// Adds a default memory of the instance's own, written as a SKILL.md, under the shipped ones.
export const NewMemoryTemplate = () => {
  const [open, setOpen] = useState(false);

  return (
    <div className="border-t border-border pt-3">
      {!open && (
        <Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)}>
          <PlusIcon className="size-4" />
          New default memory
        </Button>
      )}
      {open && (
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm font-medium">New default memory</p>
            <Button type="button" variant="ghost" size="sm" onClick={() => setOpen(false)}>
              Cancel
            </Button>
          </div>
          <InstanceTemplateEditor template={draft} onSaved={() => setOpen(false)} />
        </div>
      )}
    </div>
  );
};
