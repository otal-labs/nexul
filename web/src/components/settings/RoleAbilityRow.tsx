import { Switch } from "@/components/ui/switch";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import type { PermissionInfo } from "@/models/Permission";
import { withExtra } from "@/models/PermissionLevel";

interface RoleAbilityRowProps {
  ability: PermissionInfo;
}

// A yes-or-no permission with no ladder: the whole row is the switch's label.
export const RoleAbilityRow = ({ ability }: RoleAbilityRowProps) => {
  const { actions, setActions } = useRoleDraft();
  return (
    <li>
      <label className="flex cursor-pointer items-center gap-3 px-3 py-2.5">
        <span className="min-w-0 flex-1 text-sm font-medium">{ability.label}</span>
        <Switch
          checked={actions.includes(ability.value)}
          onCheckedChange={(on) => setActions(withExtra(actions, ability, on))}
        />
      </label>
    </li>
  );
};
