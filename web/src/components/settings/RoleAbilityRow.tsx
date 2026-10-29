import { Switch } from "@/components/ui/switch";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import type { PermissionInfo } from "@/models/Permission";
import { withExtra } from "@/models/PermissionLevel";
import { abilityLine } from "@/models/RoleSection";

interface RoleAbilityRowProps {
  ability: PermissionInfo;
}

// A yes-or-no permission with no ladder: the whole row is the switch's label.
export const RoleAbilityRow = ({ ability }: RoleAbilityRowProps) => {
  const { actions, setActions } = useRoleDraft();
  const line = abilityLine(ability.value);
  return (
    <li>
      <label className="flex cursor-pointer items-center gap-3 px-6 py-2.5">
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-medium">{ability.label}</span>
          {line && <span className="block text-xs text-muted-foreground">{line}</span>}
        </span>
        <Switch
          checked={actions.includes(ability.value)}
          onCheckedChange={(on) => setActions(withExtra(actions, ability, on))}
        />
      </label>
    </li>
  );
};
