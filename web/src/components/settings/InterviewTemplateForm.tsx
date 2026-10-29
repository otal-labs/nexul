import { useState, type FormEvent } from "react";

import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { useWorkspaceTemplateHead } from "@/components/settings/WorkspaceTemplateHead";
import { useSaveInterviewTemplate } from "@/hooks/MemoryHooks";
import { useFlash } from "@/hooks/useFlash";
import type { InterviewTemplate } from "@/models/InterviewTemplate";

export const INTERVIEW_TEMPLATE_CARD = {
  id: "interview-template",
  title: "Starting questions",
  description: "New projects' interviews start from this; existing ones never change.",
};

const FORM_ID = "interview-template-form";

interface InterviewTemplateFormProps {
  template: InterviewTemplate;
  canWrite: boolean;
}

export const InterviewTemplateForm = ({ template, canWrite }: InterviewTemplateFormProps) => {
  const [body, setBody] = useState(template.body);
  const saveTemplate = useSaveInterviewTemplate();
  const [saved, flash] = useFlash();
  const head = useWorkspaceTemplateHead({
    kind: "interview",
    name: "Interview",
    workspaceId: template.workspace_id,
    edited: template.edited,
    description: INTERVIEW_TEMPLATE_CARD.description,
    clonePermission: "memories:template",
  });

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    saveTemplate.mutate({ workspaceId: template.workspace_id, body }, { onSuccess: flash });
  };

  return (
    <SettingsCard
      {...INTERVIEW_TEMPLATE_CARD}
      stickyFooter
      description={head.description}
      aside={head.aside}
      footer={
        <GatedSaveBar
          permission="memories:template"
          verb="edit the interview template"
          form={FORM_ID}
          dirty={body !== template.body}
          saving={saveTemplate.isPending}
          saved={saved}
          onDiscard={() => setBody(template.body)}
        />
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
          hideLabel
          grow
        />
      </form>
    </SettingsCard>
  );
};
