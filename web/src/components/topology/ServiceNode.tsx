import { Handle, Position, type NodeProps } from "@xyflow/react";

import { SERVICE_W } from "@/components/topology/geometry";
import { handleClasses, nodeShell, RELATION_IN, ROUTE_IN, statusDot, wireHandle } from "@/components/topology/canvasStyles";
import { cn } from "@/lib/utils";
import type { ServiceNode as ServiceNodeType } from "@/models/Topology";

// Runner and strategy are wired in at render time (wiring.ts).
export const ServiceNode = ({ data, selected }: NodeProps<ServiceNodeType>) => (
  <div className={nodeShell} style={{ width: SERVICE_W }} data-selected={selected}>
    {/* First, so a stored relation (which names no handle) lands on the right, clear of the route wires on the left. */}
    <Handle id={RELATION_IN} type="target" position={Position.Right} className="!opacity-0" style={{ top: "70%" }} />
    <Handle id={ROUTE_IN} type="target" position={Position.Left} className={wireHandle} />
    <div className="px-3 pt-2.5 pb-2">
      <span className="block text-sm leading-5 font-medium break-words text-pretty">{data.name}</span>
      <div className="mt-1 flex items-center gap-2 text-xs leading-4 text-muted-foreground">
        <span className={cn("size-1.5 shrink-0 rounded-full", statusDot[data.status])} aria-hidden />
        <span>{data.status}</span>
        {data.strategy && <span className="ml-auto font-mono text-[11px]">{data.strategy}</span>}
      </div>
      {data.url && (
        <a className="mt-1 block font-mono text-xs break-all text-muted-foreground" href={data.url}>
          {data.url}
        </a>
      )}
    </div>
    {((data.target && !data.machineInBox) || data.replicas != null || data.volume) && (
      <div className="flex flex-col gap-0.5 border-t border-border px-3 py-1.5 font-mono text-[11px] leading-4 text-muted-foreground">
        {data.target && !data.machineInBox && <span>{data.target}</span>}
        {data.replicas != null && <span className="tabular-nums">{data.replicas} replicas</span>}
        {data.volume && <span>{data.volume}</span>}
      </div>
    )}
    <Handle type="source" position={Position.Right} className={handleClasses} style={{ top: "30%" }} />
  </div>
);
