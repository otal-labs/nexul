import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ArrowRightIcon, ShieldIcon } from "lucide-react";

import { StatusBadge } from "@/components/topology/StatusBadge";
import { cn } from "@/lib/utils";
import { ServiceStatus, type GatewayNode as GatewayNodeType } from "@/models/Topology";

// Row geometry is fixed so the layout can place each hostname pill level with its row before anything is measured.
export const GATEWAY_HEADER_H = 56;
export const GATEWAY_ROW_H = 32;
export const GATEWAY_FOOTER_H = 37;

const kindLabels = { tunnel: "Cloudflare tunnel", proxy: "Reverse proxy" } as const;

const accentStyles: Record<ServiceStatus, string> = {
  [ServiceStatus.Healthy]: "border-l-success/70",
  [ServiceStatus.Running]: "border-l-info/70",
  [ServiceStatus.Stopped]: "border-l-muted-foreground/40",
  [ServiceStatus.Failed]: "border-l-destructive/70",
};

const handleClasses = "!bg-muted-foreground";

// The hub every route passes through: one row per exposure, a wire in from its hostname pill on the left and a wire
// out to its service on the right. The row names where traffic lands, so neither wire needs a label.
export const GatewayNode = ({ data, selected }: NodeProps<GatewayNodeType>) => (
  <div
    className={cn(
      "group w-max min-w-60 rounded-lg border border-border border-l-[3px] bg-card shadow-card ring-0 transition-[transform,box-shadow] duration-150 ease-standard hover:-translate-y-0.5 hover:shadow-elevated",
      accentStyles[data.status],
      selected && "ring-2 ring-ring",
    )}
    data-selected={selected}
  >
    <div className="px-3 py-2.5" style={{ height: GATEWAY_HEADER_H }}>
      <div className="flex items-center gap-2">
        <ShieldIcon className="size-4 shrink-0" aria-hidden />
        <span className="whitespace-nowrap text-sm font-medium">{kindLabels[data.kind]}</span>
      </div>
      <span className="block whitespace-nowrap font-mono text-xs text-muted-foreground">{data.name}</span>
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
    <div className="flex items-center justify-between gap-4 border-t border-border px-3 py-2">
      <StatusBadge status={data.status} />
      <span className="whitespace-nowrap font-mono text-xs text-muted-foreground">
        {[data.target, ...data.networks].filter(Boolean).join(" · ")}
      </span>
    </div>
  </div>
);
