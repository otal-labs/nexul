import { errorMessage } from "@/api/client";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface InterviewTemplateFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  questionCount: number;
  error: Error | null;
  readOnly?: boolean;
  // The page around it already names the field and counts its questions; the label stays for screen readers.
  hideLabel?: boolean;
  // Grows with its text instead of scrolling inside the page's own scroll.
  grow?: boolean;
}

export const InterviewTemplateField = ({ id, value, onChange, questionCount, error, readOnly = false, hideLabel = false, grow = false }: InterviewTemplateFieldProps) => (
  <div className="space-y-2">
    <div className={cn("flex items-baseline justify-between gap-3", hideLabel && "sr-only")}>
      <label htmlFor={id} className="text-sm font-medium">
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
      className={cn("min-h-80 bg-surface-2 font-mono text-xs leading-relaxed", grow && "field-sizing-content resize-none")}
    />
    {error && (
      <p role="alert" className="font-mono text-xs text-destructive">
        {errorMessage(error)}
      </p>
    )}
  </div>
);
