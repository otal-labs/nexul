import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import type { DiscoveredContainer } from "@/models/Machine";

interface ImportContainerRowProps {
  container: DiscoveredContainer;
  checked: boolean;
  onToggle: () => void;
  // Under a compose project's heading, its checkbox in from the project's own.
  indent?: boolean;
}

export const ImportContainerRow = ({ container, checked, onToggle, indent = false }: ImportContainerRowProps) => (
  <label className={cn("flex cursor-pointer items-center gap-3 px-4 py-3 text-sm", indent && "py-2 pl-11")}>
    <Checkbox checked={checked} onCheckedChange={onToggle} />
    <span className="min-w-0 truncate" title={container.name}>
      {container.name}
    </span>
    <span className="min-w-0 font-mono text-xs wrap-anywhere text-muted-foreground">{container.image}</span>
  </label>
);
