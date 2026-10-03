import { useState, type FormEvent } from "react";

import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useSaveInterviewTemplate } from "@/hooks/MemoryHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import type { InterviewTemplate } from "@/models/InterviewTemplate";

interface InterviewTemplateFormProps {
  template: InterviewTemplate;
  canWrite: boolean;
}

export const InterviewTemplateForm = ({ template, canWrite }: InterviewTemplateFormProps) => {
  const [body, setBody] = useState(template.body);
  const saveTemplate = useSaveInterviewTemplate();
  const openClone = useCloneTemplateDialog();
  const at = { scope: "workspace" as const, workspace_id: template.workspace_id };

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    saveTemplate.mutate({ workspaceId: template.workspace_id, body });
  };

  return (
    <form onSubmit={onSubmit} className="space-y-3">
      <TemplateOriginLine
        kind="interview"
        templateKey=""
        at={at}
        state={template.edited ? "edited" : "following"}
        canReset={canWrite}
      />
      <InterviewTemplateField
        id="interview-template-body"
        value={body}
        onChange={setBody}
        questionCount={template.questions.length}
        error={saveTemplate.error}
        readOnly={!canWrite}
      />
      <div className="flex flex-wrap items-center justify-end gap-2">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => void openClone({ kind: "interview", key: "", name: "Interview", from: at })}
        >
          Clone to…
        </Button>
        {canWrite && (
          <Button type="submit" size="sm" loading={saveTemplate.isPending} disabled={body === template.body}>
            Save
          </Button>
        )}
      </div>
    </form>
  );
};
