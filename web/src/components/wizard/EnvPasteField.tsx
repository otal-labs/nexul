import { Textarea } from "@/components/ui/textarea";
import type { ParsedEnvFile } from "@/models/EnvFile";

interface EnvPasteFieldProps {
  text: string;
  onText: (text: string) => void;
  parsed: ParsedEnvFile;
}

// The values are secrets: no spellcheck or autofill, and the text never leaves this component except through Save.
export const EnvPasteField = ({ text, onText, parsed }: EnvPasteFieldProps) => (
  <div className="space-y-2">
    <Textarea
      aria-label="Environment file"
      aria-invalid={parsed.invalidLines.length > 0}
      aria-describedby="env-paste-notes"
      value={text}
      onChange={(e) => onText(e.target.value)}
      spellCheck={false}
      autoComplete="off"
      autoCapitalize="off"
      autoCorrect="off"
      rows={10}
      placeholder="KEY=value"
      className="font-mono text-xs"
    />
    <div id="env-paste-notes" className="space-y-1 text-xs">
      {parsed.invalidLines.map((line) => (
        <p key={line} className="text-destructive">
          Line {line} isn&apos;t KEY=value
        </p>
      ))}
      {parsed.duplicateKeys.length > 0 && (
        <p className="text-muted-foreground">
          {parsed.duplicateKeys.join(", ")} appears more than once. The last value wins.
        </p>
      )}
    </div>
  </div>
);
