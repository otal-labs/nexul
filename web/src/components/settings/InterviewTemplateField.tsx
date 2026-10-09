import { errorMessage } from "@/api/client";
import { Textarea } from "@/components/ui/textarea";

interface InterviewTemplateFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  questionCount: number;
  error: Error | null;
  readOnly?: boolean;
}

export const InterviewTemplateField = ({ id, value, onChange, questionCount, error, readOnly = false }: InterviewTemplateFieldProps) => (
  <div className="space-y-2">
    <div className="flex items-baseline justify-between gap-3">
      <label htmlFor={id} className="text-xs font-medium text-muted-foreground">
        Template (markdown)
      </label>
      <span className="font-mono text-xs tabular-nums text-muted-foreground">
        {questionCount} {questionCount === 1 ? "question" : "questions"}
      </span>
    </div>
    <Textarea
      id={id}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      rows={16}
      spellCheck={false}
      className="min-h-80 bg-surface-2 font-mono text-xs leading-relaxed"
    />
    {error && (
      <p role="alert" className="font-mono text-xs text-destructive">
        {errorMessage(error)}
      </p>
    )}
  </div>
);
