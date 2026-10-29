import { Checkbox } from "@/components/ui/checkbox";
import { TargetMark } from "@/components/access/TargetMark";
import { wizardCardClass } from "@/components/wizard/WizardCardList";
import type { PanelTarget } from "@/hooks/usePanelTargets";

interface WizardRoleRowProps {
  role: PanelTarget;
  checked: boolean;
  line: string;
  // The viewer may not change who is in: the tick stays as it is, announced as unavailable.
  locked: boolean;
  onToggle: (checked: boolean) => void;
}

export const WizardRoleRow = ({ role, checked, line, locked, onToggle }: WizardRoleRowProps) => (
  <li className={wizardCardClass}>
    <label className="flex cursor-pointer items-start gap-3 px-4 py-3 has-[[aria-disabled=true]]:cursor-default">
      <Checkbox
        checked={checked}
        aria-disabled={locked}
        onCheckedChange={(v) => !locked && onToggle(v === true)}
        aria-label={role.name}
        className="mt-0.5 aria-disabled:opacity-60"
      />
      <TargetMark target={role} className="-mt-0.5" />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{role.name}</span>
        <span className="block truncate text-xs text-muted-foreground">{line}</span>
      </span>
    </label>
  </li>
);
