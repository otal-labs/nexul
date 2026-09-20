import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import type { DiscoveredContainer } from "@/models/Machine";

interface ImportContainerRowProps {
  container: DiscoveredContainer;
  checked: boolean;
  onToggle: () => void;
  // Nested under a compose-project heading vs. a top-level standalone/gateway row.
  indent?: boolean;
}

export const ImportContainerRow = ({ container, checked, onToggle, indent = false }: ImportContainerRowProps) => (
  <label className={cn("flex items-center gap-3 py-1.5 text-sm", indent && "pl-6")}>
    <Checkbox checked={checked} onCheckedChange={onToggle} />
    <span className="min-w-0 truncate">{container.name}</span>
    <span className="truncate font-mono text-xs text-muted-foreground">{container.image}</span>
  </label>
);
