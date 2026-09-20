import { LockIcon } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import type { Memory } from "@/models/Memory";

interface MemoryPickRowProps {
  memory: Memory;
  checked: boolean;
  onToggle: () => void;
}

// An always-included memory is ticked and locked: the run inlines it whether or not the caller asks.
export const MemoryPickRow = ({ memory, checked, onToggle }: MemoryPickRowProps) => (
  <label className="flex cursor-pointer items-start gap-3 border-b border-border px-3 py-2 last:border-b-0 hover:bg-accent/40">
    <Checkbox
      checked={checked}
      disabled={memory.always_included}
      onCheckedChange={onToggle}
      className="mt-0.5"
      aria-label={memory.title}
    />
    <span className="min-w-0 flex-1">
      <span className="flex items-center gap-1.5 text-sm font-medium">
        {memory.title}
        {memory.always_included && <LockIcon className="size-3 text-muted-foreground" role="img" aria-label="Always included" />}
      </span>
      {memory.when_to_use !== "" && (
        <span className="block truncate font-mono text-[11px] text-muted-foreground">{memory.when_to_use}</span>
      )}
    </span>
  </label>
);
