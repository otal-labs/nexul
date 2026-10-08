import { Microheader } from "@/components/Microheader";
import { PermissionLevels } from "@/components/access/PermissionLevels";
import { PermissionArea, type PermissionInfo } from "@/models/Permission";

interface RoleLevelSectionsProps {
  catalog: PermissionInfo[];
  value: string[];
  onChange: (value: string[]) => void;
}

// A role's levels split by where each area applies, from the catalog's area; nothing here names a domain.
export const RoleLevelSections = ({ catalog, value, onChange }: RoleLevelSectionsProps) => {
  const workspace = catalog.filter((entry) => entry.area !== PermissionArea.Project);
  const project = catalog.filter((entry) => entry.area === PermissionArea.Project);
  return (
    <div className="space-y-5">
      {workspace.length > 0 && (
        <section className="space-y-2">
          <Microheader>Workspace</Microheader>
          <PermissionLevels entries={workspace} value={value} onChange={onChange} />
        </section>
      )}
      {project.length > 0 && (
        <section className="space-y-2">
          <Microheader>Every project</Microheader>
          <p className="text-sm text-muted-foreground">Applies to members whose Every project is From role.</p>
          <PermissionLevels entries={project} value={value} onChange={onChange} everyLabel="Every area" />
        </section>
      )}
    </div>
  );
};
