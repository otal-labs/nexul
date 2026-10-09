import { useState, type FormEvent } from "react";

import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useSaveInterviewTemplate } from "@/hooks/MemoryHooks";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useFlash } from "@/hooks/useFlash";
import type { InterviewTemplate } from "@/models/InterviewTemplate";

export const INTERVIEW_TEMPLATE_CARD = {
  id: "interview-template",
  title: "Interview template",
  description:
    "Every new project's interview starts from this. It follows the instance template until this workspace saves its own. Editing it never changes an existing interview.",
};

const FORM_ID = "interview-template-form";

interface InterviewTemplateFormProps {
  template: InterviewTemplate;
  canWrite: boolean;
}

export const InterviewTemplateForm = ({ template, canWrite }: InterviewTemplateFormProps) => {
  const [body, setBody] = useState(template.body);
  const saveTemplate = useSaveInterviewTemplate();
  const openClone = useCloneTemplateDialog();
  const [saved, flash] = useFlash();
  const at = { scope: "workspace" as const, workspace_id: template.workspace_id };

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    saveTemplate.mutate({ workspaceId: template.workspace_id, body }, { onSuccess: flash });
  };

  return (
    <SettingsCard
      {...INTERVIEW_TEMPLATE_CARD}
      aside={
        <TemplateOriginLine kind="interview" templateKey="" at={at} state={template.edited ? "edited" : "following"} canReset={canWrite} />
      }
      footer={
        canWrite && (
          <SettingsSaveBar
            form={FORM_ID}
            dirty={body !== template.body}
            saving={saveTemplate.isPending}
            saved={saved}
            onDiscard={() => setBody(template.body)}
          />
        )
      }
    >
      <form id={FORM_ID} onSubmit={onSubmit} className="space-y-3">
        <InterviewTemplateField
          id="interview-template-body"
          value={body}
          onChange={setBody}
          questionCount={template.questions.length}
          error={saveTemplate.error}
          readOnly={!canWrite}
        />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2.5"
          onClick={() => void openClone({ kind: "interview", key: "", name: "Interview", from: at })}
        >
          Clone to…
        </Button>
      </form>
    </SettingsCard>
  );
};
