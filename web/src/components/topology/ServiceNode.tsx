import { Handle, Position, type NodeProps } from "@xyflow/react";

import { RuntimeIcon } from "@/components/topology/RuntimeIcon";
import { StatusBadge } from "@/components/topology/StatusBadge";
import { cn } from "@/lib/utils";
import type { ServiceNode as ServiceNodeType } from "@/models/Topology";

const handleClasses =
  "!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125";

// Every card on the canvas lifts 2px on hover and takes the brand ring when selected.
export const nodeShell =
  "group canvas-card w-max transition-[translate] duration-150 ease-standard hover:-translate-y-0.5 data-[selected=true]:outline-2 data-[selected=true]:outline-ring";

const Meta = ({ data }: { data: ServiceNodeType["data"] }) => (
  <>
    {data.target && (
      <span>
        {data.target}
        {data.strategy && ` · ${data.strategy}`}
      </span>
    )}
    {data.replicas != null && <span className="tabular-nums">{data.replicas} replicas</span>}
    {data.volume && <span>{data.volume}</span>}
  </>
);

// Name and status come from the anchored service definition; runner and strategy are wired in at render time.
// Hover/select feedback and handle-grow are plain CSS so only the hovered node repaints during drag/connect.
export const ServiceNode = ({ data, selected }: NodeProps<ServiceNodeType>) => (
  <div className={cn(nodeShell, "min-w-60")} data-selected={selected}>
    <Handle type="target" position={Position.Left} className={handleClasses} />
    <div className="flex items-center gap-2.5 px-3 py-2.5">
      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-surface-2 text-muted-foreground">
        <RuntimeIcon runtime={data.runtime ?? "docker"} className="size-3.5" />
      </span>
      <div className="min-w-0 flex-1">
        <span className="block whitespace-nowrap text-sm font-medium">{data.name}</span>
        {data.url && (
          <a className="block whitespace-nowrap font-mono text-xs text-muted-foreground" href={data.url}>
            {data.url}
          </a>
        )}
      </div>
      <StatusBadge status={data.status} className="ml-3" />
    </div>
    {(data.target || data.replicas != null || data.volume) && (
      <div className="flex flex-col gap-0.5 whitespace-nowrap border-t border-border px-3 py-2 font-mono text-[11px] text-muted-foreground">
        <Meta data={data} />
      </div>
    )}
    <Handle type="source" position={Position.Right} className={handleClasses} />
  </div>
);

