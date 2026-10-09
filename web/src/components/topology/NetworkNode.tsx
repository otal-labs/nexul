import { Handle, Position, type NodeProps } from "@xyflow/react";
import { NetworkIcon } from "lucide-react";

import { nodeShell } from "@/components/topology/ServiceNode";
import { cn } from "@/lib/utils";
import type { NetworkNode as NetworkNodeType } from "@/models/Topology";

// Documents which containers share a network via its edges; real membership comes from compose files.
export const NetworkNode = ({ data, selected }: NodeProps<NetworkNodeType>) => (
  <div
    className={cn(
      nodeShell,
      "min-w-48 p-3",
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
      <span className="whitespace-nowrap text-sm font-medium">{data.name}</span>
    </div>
    <span className="font-mono text-xs text-muted-foreground">network</span>
    <Handle
      type="source"
      position={Position.Right}
      className="!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125"
    />
  </div>
);
