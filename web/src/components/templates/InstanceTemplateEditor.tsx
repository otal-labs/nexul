import { useState } from "react";

import { TemplateBodyField } from "@/components/templates/TemplateBodyField";
import { Button } from "@/components/ui/button";
import { useResetInstanceTemplate, useSaveInstanceTemplate } from "@/hooks/TemplateHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { BELOW_SCOPE, INSTANCE, type Template } from "@/models/Template";

interface InstanceTemplateEditorProps {
  template: Template;
}

export const InstanceTemplateEditor = ({ template }: InstanceTemplateEditorProps) => {
  const [body, setBody] = useState(template.body);
  const save = useSaveInstanceTemplate();
  const reset = useResetInstanceTemplate();
  const openClone = useCloneTemplateDialog();
  const { open: confirm } = useConfirmationDialog();
  const { kind, key, name } = template;

  const onReset = async () => {
    const ok = await confirm({
      title: "Reset to the default?",
      message: `The instance's ${name} template goes back to the text Nexul ships with.`,
      confirmLabel: "Reset",
    });
    if (ok) reset.mutate({ kind, key });
  };

  return (
    <div className="animate-in space-y-3 pt-3 fade-in-0 slide-in-from-top-1 duration-200 ease-out">
      <TemplateBodyField
        kind={kind}
        id={`instance-template-${kind}-${key}`}
        value={body}
        onChange={setBody}
        questionCount={template.questions?.length ?? 0}
        error={save.error}
      />
      <div className="flex flex-wrap items-center justify-end gap-2">
        {BELOW_SCOPE[kind] && (
          <Button type="button" variant="ghost" size="sm" onClick={() => void openClone({ kind, key, name, from: INSTANCE })}>
            Clone to…
          </Button>
        )}
        <Button type="button" variant="ghost" size="sm" disabled={!template.edited} loading={reset.isPending} onClick={() => void onReset()}>
          Reset to default
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={body === template.body}
          loading={save.isPending}
          onClick={() => save.mutate({ kind, key, body })}
        >
          Save
        </Button>
      </div>
    </div>
  );
};
