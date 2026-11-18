import { Input } from "@/components/ui/input";
import { SettingsRow } from "@/components/settings/SettingsRow";
import type { ImportField } from "@/models/Import";

interface ImportTextRowProps {
  id: string;
  field: ImportField;
  value: string;
  onChange: (value: string) => void;
  secret?: boolean;
}

// One text field a source asks for; a secret one is masked and never filled in by the browser.
export const ImportTextRow = ({ id, field, value, onChange, secret }: ImportTextRowProps) => (
  <SettingsRow label={field.label} htmlFor={id} description={field.hint}>
    <Input
      id={id}
      type={secret ? "password" : "text"}
      autoComplete={secret ? "off" : undefined}
      spellCheck={false}
      placeholder={field.placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  </SettingsRow>
);
