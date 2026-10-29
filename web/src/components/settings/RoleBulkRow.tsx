import { PermissionLevelControl } from "@/components/access/PermissionLevelControl";
import { useRoleDraft } from "@/hooks/useRoleDraft";
import { uniformLevelOf, withLevelEverywhere, type PermissionDomain } from "@/models/PermissionLevel";

interface RoleBulkRowProps {
  label: string;
  areas: PermissionDomain[];
}

// Sets every area row of its tab at once and selects no rung once they differ; it never touches a switch.
export const RoleBulkRow = ({ label, areas }: RoleBulkRowProps) => {
  const { actions, setActions } = useRoleDraft();
  return (
    <li className="flex items-center gap-3 bg-muted/40 px-6 py-2 dark:bg-muted/80">
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{label}</span>
        <span className="block truncate text-xs text-muted-foreground">Sets every row below at once</span>
      </span>
      <PermissionLevelControl
        label={`${label} access`}
        levelCount={Math.max(0, ...areas.map((domain) => domain.levels.length))}
        level={uniformLevelOf(areas, actions)}
        onLevel={(level) => setActions(withLevelEverywhere(actions, areas, level))}
      />
    </li>
  );
};
