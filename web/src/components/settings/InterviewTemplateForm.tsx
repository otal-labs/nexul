import { useState, type FormEvent } from "react";

import { InterviewLengthMeter } from "@/components/memory/InterviewLengthMeter";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useSaveInterviewTemplate } from "@/hooks/MemoryHooks";
import type { InterviewTemplate } from "@/models/InterviewTemplate";

interface InterviewTemplateFormProps {
  template: InterviewTemplate;
  canWrite: boolean;
}

export const InterviewTemplateForm = ({ template, canWrite }: InterviewTemplateFormProps) => {
  const [body, setBody] = useState(template.body);
  const saveTemplate = useSaveInterviewTemplate();

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    saveTemplate.mutate({ workspaceId: template.workspace_id, body });
  };

  return (
    <form onSubmit={onSubmit} className="space-y-3">
      <label htmlFor="interview-template-body" className="block text-xs font-medium text-muted-foreground">
        Template (markdown)
      </label>
      <Textarea
        id="interview-template-body"
        value={body}
        onChange={(e) => setBody(e.target.value)}
        readOnly={!canWrite}
        rows={16}
        spellCheck={false}
        className="font-mono text-xs"
      />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <InterviewLengthMeter length={body.length} />
        {canWrite && (
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={body === template.default_body}
              onClick={() => setBody(template.default_body)}
            >
              Reset to default
            </Button>
            <Button type="submit" size="sm" disabled={body === template.body || saveTemplate.isPending}>
              Save
            </Button>
          </div>
        )}
      </div>
    </form>
  );
};
