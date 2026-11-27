import { SettingsRow } from "@/components/settings/SettingsRow";
import { Switch } from "@/components/ui/switch";

interface ComputerSwitchRowProps {
  label: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onChange: (on: boolean) => void;
}

// One on/off setting of a computer; it saves the moment it is flipped.
export const ComputerSwitchRow = ({ label, description, checked, disabled, onChange }: ComputerSwitchRowProps) => (
  <SettingsRow label={label} description={description} compact className="last:pb-4">
    <Switch aria-label={label} checked={checked} disabled={disabled} onCheckedChange={onChange} />
  </SettingsRow>
);
