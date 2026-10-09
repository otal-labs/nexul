import { useState, type FormEvent } from "react";

import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useSaveInterviewTemplate } from "@/hooks/MemoryHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import type { InterviewTemplate } from "@/models/InterviewTemplate";

export const INTERVIEW_TEMPLATE_CARD = {
  id: "interview-template",
  title: "Interview template",
  description:
    "Every new project's interview starts from this. It follows the instance template until this workspace saves its own. Editing it never changes an existing interview.",
};

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
    <SettingsCard
      {...INTERVIEW_TEMPLATE_CARD}
      footer={
        <>
          <TemplateOriginLine
            kind="interview"
            templateKey=""
            at={at}
            state={template.edited ? "edited" : "following"}
            canReset={canWrite}
          />
          <div className="ml-auto flex items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => void openClone({ kind: "interview", key: "", name: "Interview", from: at })}
            >
              Clone to…
            </Button>
            {canWrite && (
              <Button type="submit" form="interview-template-form" size="sm" loading={saveTemplate.isPending} disabled={body === template.body}>
                Save
              </Button>
            )}
          </div>
        </>
      }
    >
      <form id="interview-template-form" onSubmit={onSubmit}>
        <InterviewTemplateField
          id="interview-template-body"
          value={body}
          onChange={setBody}
          questionCount={template.questions.length}
          error={saveTemplate.error}
          readOnly={!canWrite}
        />
      </form>
    </SettingsCard>
  );
};
