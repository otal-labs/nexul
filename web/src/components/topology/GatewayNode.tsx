import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ArrowRightIcon, ShieldIcon } from "lucide-react";

import { nodeShell } from "@/components/topology/ServiceNode";
import { StatusBadge } from "@/components/topology/StatusBadge";
import { cn } from "@/lib/utils";
import type { GatewayNode as GatewayNodeType } from "@/models/Topology";

// Row geometry is fixed so the layout can place each hostname pill level with its row before anything is measured.
export const GATEWAY_HEADER_H = 56;
export const GATEWAY_ROW_H = 32;
export const GATEWAY_FOOTER_H = 37;

const kindLabels = { tunnel: "Cloudflare tunnel", proxy: "Reverse proxy" } as const;

const handleClasses = "!bg-muted-foreground";

// The hub every route passes through: one row per exposure, a wire in from its hostname pill on the left and a wire
// out to its service on the right. The row names where traffic lands, so neither wire needs a label.
export const GatewayNode = ({ data, selected }: NodeProps<GatewayNodeType>) => (
  <div className={cn(nodeShell, "min-w-60")} data-selected={selected}>
    <div className="flex items-center gap-2.5 px-3" style={{ height: GATEWAY_HEADER_H }}>
      <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-surface-2 text-muted-foreground">
        <ShieldIcon className="size-3.5" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <span className="block whitespace-nowrap text-sm font-medium">{kindLabels[data.kind]}</span>
        <span className="block whitespace-nowrap font-mono text-xs text-muted-foreground">{data.name}</span>
      </div>
      <StatusBadge status={data.status} className="ml-3" />
    </div>
    <ul className="divide-y divide-border border-t border-border">
      {data.routes.map((r) => (
        <li
          key={r.id}
          className="relative flex items-center gap-2 whitespace-nowrap px-3 font-mono text-xs"
          style={{ height: GATEWAY_ROW_H }}
          data-route={r.id}
        >
          <Handle id={`in-${r.id}`} type="target" position={Position.Left} className={handleClasses} style={{ top: "50%" }} />
          <ArrowRightIcon className="size-3 shrink-0 text-muted-foreground" aria-hidden />
          <span>
            {r.service}:{r.port}
          </span>
          {r.address && (
            <span className="text-muted-foreground">
              ({r.address}:{r.port})
            </span>
          )}
          <Handle id={r.id} type="source" position={Position.Right} className={handleClasses} style={{ top: "50%" }} />
        </li>
      ))}
    </ul>
    <div className="whitespace-nowrap border-t border-border px-3 py-2 font-mono text-[11px] leading-5 text-muted-foreground">
      {[data.target, ...data.networks].filter(Boolean).join(" · ")}
    </div>
  </div>
);
