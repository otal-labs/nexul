import { Checkbox } from "@/components/ui/checkbox";
import { TargetMark } from "@/components/access/TargetMark";
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
  <li>
    <label className="flex cursor-pointer items-center gap-3 py-2.5 has-[[aria-disabled=true]]:cursor-default">
      <Checkbox
        checked={checked}
        aria-disabled={locked}
        onCheckedChange={(v) => !locked && onToggle(v === true)}
        aria-label={role.name}
        className="aria-disabled:opacity-60"
      />
      <TargetMark target={role} />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{role.name}</span>
        <span className="block truncate text-xs text-muted-foreground">{line}</span>
      </span>
    </label>
  </li>
);
