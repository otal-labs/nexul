import { InterviewLengthMeter } from "@/components/memory/InterviewLengthMeter";
import { Textarea } from "@/components/ui/textarea";

interface InterviewTemplateFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

export const InterviewTemplateField = ({ id, value, onChange, readOnly = false }: InterviewTemplateFieldProps) => (
  <div className="space-y-3">
    <label htmlFor={id} className="block text-xs font-medium text-muted-foreground">
      Template (markdown)
    </label>
    <Textarea
      id={id}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      rows={16}
      spellCheck={false}
      className="font-mono text-xs"
    />
    <InterviewLengthMeter length={value.length} />
  </div>
);
