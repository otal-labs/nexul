import { Handle, Position, type NodeProps } from "@xyflow/react";
import { NetworkIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import type { NetworkNode as NetworkNodeType } from "@/models/Topology";

// Documents which containers share a network via its edges; real membership comes from compose files.
export const NetworkNode = ({ data, selected }: NodeProps<NetworkNodeType>) => (
  <div
    className={cn(
      "group w-48 rounded-lg border border-border bg-card p-3 shadow-card ring-0 transition-[transform,box-shadow,border-color] duration-150 ease-standard hover:-translate-y-0.5 hover:border-ring/40 hover:shadow-elevated",
      selected && "ring-2 ring-ring",
    )}
    data-selected={selected}
  >
    <Handle
      type="target"
      position={Position.Left}
      className="!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125"
    />
    <div className="flex items-center gap-2">
      <NetworkIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="truncate text-sm font-medium">{data.name}</span>
    </div>
    <span className="font-mono text-xs text-muted-foreground">network</span>
    <Handle
      type="source"
      position={Position.Right}
      className="!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125"
    />
  </div>
);
