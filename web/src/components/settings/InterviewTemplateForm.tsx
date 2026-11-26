import { useState, type FormEvent } from "react";

import { errorMessage } from "@/api/client";
import { InterviewQuestionList } from "@/components/settings/InterviewQuestionList";
import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
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
  const [view, setView] = useState<"questions" | "markdown">("questions");
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
        <ToggleGroup
          type="single"
          variant="segmented"
          size="xs"
          aria-label="Template view"
          value={view}
          onValueChange={(next) => next && setView(next as typeof view)}
        >
          <ToggleGroupItem value="questions">Questions</ToggleGroupItem>
          <ToggleGroupItem value="markdown">Markdown</ToggleGroupItem>
        </ToggleGroup>
        {view === "questions" && <InterviewQuestionList body={body} onChange={setBody} readOnly={!canWrite} />}
        {view === "questions" && saveTemplate.error && (
          <p role="alert" className="font-mono text-xs text-destructive">
            {errorMessage(saveTemplate.error)}
          </p>
        )}
        {view === "markdown" && (
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
        )}
      </form>
    </SettingsCard>
  );
};
