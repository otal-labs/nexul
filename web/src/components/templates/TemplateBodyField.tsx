import { InterviewTemplateField } from "@/components/settings/InterviewTemplateField";
import { MentionChipField } from "@/components/settings/MentionChipField";
import { Textarea } from "@/components/ui/textarea";
import type { TemplateKind } from "@/models/Template";

interface TemplateBodyFieldProps {
  kind: TemplateKind;
  id: string;
  value: string;
  onChange: (value: string) => void;
}

// The same field each kind is edited with in a workspace or project, so the instance editor reads the same.
export const TemplateBodyField = ({ kind, id, value, onChange }: TemplateBodyFieldProps) => (
  <>
    {kind === "interview" && <InterviewTemplateField id={id} value={value} onChange={onChange} />}
    {kind === "mention_chip" && <MentionChipField id={id} value={value} onChange={onChange} />}
    {kind === "play_instructions" && (
      <div className="space-y-2">
        <label htmlFor={id} className="text-sm font-medium">
          Instructions
        </label>
        <Textarea id={id} value={value} onChange={(e) => onChange(e.target.value)} rows={6} placeholder="What the Agent should do on this run" />
      </div>
    )}
    {kind === "ticket_body" && (
      <div className="space-y-1.5">
        <label htmlFor={id} className="text-sm font-medium">
          Template (markdown)
        </label>
        <Textarea
          id={id}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          rows={12}
          placeholder={"## Why\n\n## Acceptance criteria"}
          className="font-mono text-xs"
        />
      </div>
    )}
  </>
);
