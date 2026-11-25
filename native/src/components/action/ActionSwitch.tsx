import { Switch } from "react-native";
import { useCSSVariable } from "uniwind";

import type { ActionGate } from "@nexul/client-core/actionState";

import { useActionState } from "@/hooks/useActionState";

interface ActionSwitchProps extends ActionGate {
  label: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
}

// A viewer who can't flip it still sees the state, switched off from touch, with the reason as its hint when their role shows locked actions.
export const ActionSwitch = ({ permission, projectId, answer, verb, label, value, onValueChange, disabled = false }: ActionSwitchProps) => {
  const action = useActionState({ permission, projectId, answer, verb });
  const [brand, offTrack] = useCSSVariable(["--color-brand", "--color-muted-foreground"]);
  return (
    <Switch
      accessibilityLabel={label}
      accessibilityHint={action.state === "locked" ? action.reason : undefined}
      value={value}
      onValueChange={onValueChange}
      disabled={action.state !== "live" || disabled}
      trackColor={{ true: String(brand), false: String(offTrack) }}
      ios_backgroundColor={String(offTrack)}
    />
  );
};
