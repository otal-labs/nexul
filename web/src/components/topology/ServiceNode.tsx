import { Handle, Position, type NodeProps } from "@xyflow/react";

import { RuntimeIcon } from "@/components/topology/RuntimeIcon";
import { StatusBadge } from "@/components/topology/StatusBadge";
import { cn } from "@/lib/utils";
import {
  ServiceStatus,
  type ServiceNode as ServiceNodeType,
} from "@/models/Topology";

// Status accent on the node's leading edge uses the same hue language as the status badges.
const accentStyles: Record<ServiceStatus, string> = {
  [ServiceStatus.Healthy]: "border-l-success/70",
  [ServiceStatus.Running]: "border-l-info/70",
  [ServiceStatus.Stopped]: "border-l-muted-foreground/40",
  [ServiceStatus.Failed]: "border-l-destructive/70",
};

const handleClasses =
  "!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125";

// Name/status come from the anchored service definition; runner and strategy are wired in at render time.
// Hover/select feedback and handle-grow are plain CSS so only the hovered node repaints during drag/connect.
export const ServiceNode = ({ data, selected }: NodeProps<ServiceNodeType>) => (
  <div
    className={cn(
      "group w-max min-w-64 rounded-lg border border-border border-l-[3px] bg-card shadow-card ring-0 transition-[transform,box-shadow,border-color] duration-150 ease-standard hover:-translate-y-0.5 hover:shadow-elevated",
      accentStyles[data.status],
      selected && "ring-2 ring-ring",
    )}
    data-selected={selected}
  >
    <Handle type="target" position={Position.Left} className={handleClasses} />
    <div className="px-3 py-2.5">
      <div className="flex items-center gap-2">
        <RuntimeIcon runtime={data.runtime ?? "docker"} />
        <span className="whitespace-nowrap text-sm font-medium">
          {data.name}
        </span>
      </div>
      {data.url && (
        <a
          className="block whitespace-nowrap font-mono text-xs text-muted-foreground"
          href={data.url}
        >
          {data.url}
        </a>
      )}
    </div>
    <div className="flex items-center justify-between gap-2 border-t border-border px-3 py-2">
      <StatusBadge status={data.status} />
      {data.replicas != null && (
        <span className="font-mono text-xs tabular-nums text-muted-foreground">
          {data.replicas} replicas
        </span>
      )}
    </div>
    {data.target && (
      <div className="whitespace-nowrap border-t border-border px-3 py-2 font-mono text-xs text-muted-foreground">
        {data.target}
        {data.strategy && ` · ${data.strategy}`}
      </div>
    )}
    {data.volume && (
      <div className="whitespace-nowrap border-t border-border px-3 py-2 font-mono text-xs text-muted-foreground">
        {data.volume}
      </div>
    )}
    <Handle type="source" position={Position.Right} className={handleClasses} />
  </div>
);
