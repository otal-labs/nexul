import { Switch } from "@/components/ui/switch";
import { useRoleDraft } from "@/hooks/useRoleDraft";

// The role's Show locked actions switch: not a permission, so it sits beside them as a row of its own.
export const RoleLockedActionsRow = () => {
  const { showLocked, setShowLocked } = useRoleDraft();
  return (
    <li>
      <label className="flex cursor-pointer items-center gap-3 px-6 py-2.5">
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-medium">Show locked actions</span>
          <span className="block text-xs text-muted-foreground">
            Holders see an action they can't take dimmed, with the reason, instead of not seeing it.
          </span>
        </span>
        <Switch checked={showLocked} onCheckedChange={setShowLocked} />
      </label>
    </li>
  );
};
