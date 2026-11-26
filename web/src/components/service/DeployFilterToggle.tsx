import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useDeployFilter, type DeployFilter } from "@/hooks/useDeployFilter";

export const DeployFilterToggle = () => {
  const [filter, setFilter] = useDeployFilter();
  return (
    <ToggleGroup type="single" variant="segmented" size="xs" value={filter} onValueChange={(value) => value && setFilter(value as DeployFilter)} aria-label="Show">
      <ToggleGroupItem value="all">All</ToggleGroupItem>
      <ToggleGroupItem value="failed">Failed</ToggleGroupItem>
    </ToggleGroup>
  );
};
