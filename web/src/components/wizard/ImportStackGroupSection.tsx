import { ImportContainerRow } from "@/components/wizard/ImportContainerRow";
import type { StackGroup } from "@/models/Machine";

interface ImportStackGroupSectionProps {
  group: StackGroup;
  selected: Set<string>;
  onToggleContainer: (name: string) => void;
}

// One compose project's containers (spec §8: "grouped by compose project label"); adopted as one unmanaged
// stack, slug = the project name.
export const ImportStackGroupSection = ({ group, selected, onToggleContainer }: ImportStackGroupSectionProps) => (
  <div className="space-y-1">
    <p className="font-mono text-sm font-medium">{group.project}</p>
    {group.containers.map((container) => (
      <ImportContainerRow
        key={container.name}
        container={container}
        checked={selected.has(container.name)}
        onToggle={() => onToggleContainer(container.name)}
        indent
      />
    ))}
  </div>
);
