import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ShieldIcon } from "lucide-react";

import { footerFacts, footerOnOneLine, GATEWAY_W, gatewayHeaderHeight, routeAddress, routeLines } from "@/components/topology/geometry";
import { nodeShell, statusDot, toneText, wireHandle } from "@/components/topology/canvasStyles";
import { cn } from "@/lib/utils";
import type { GatewayNode as GatewayNodeType, RouteRow } from "@/models/Topology";

const kindLabels = { tunnel: "Cloudflare tunnel", proxy: "Reverse proxy" } as const;


const RouteItem = ({ route, height, dim }: { route: RouteRow; height: number | undefined; dim: boolean }) => {
  const { address: ownLine } = routeLines(route);
  const name = (
    <span className="min-w-0 flex-1 pl-[2ch] -indent-[2ch]">
      <span className={cn("relative -top-px text-muted-foreground", route.tone && toneText[route.tone])}>→ </span>
      {route.service}
      <span className="text-muted-foreground">:{route.port}</span>
    </span>
  );
  const address = route.address && <span className="shrink-0 text-right tabular-nums text-muted-foreground">{routeAddress(route)}</span>;
  // The words sit centred in the row, which may have grown to its pill's height, so they stay level with the wire.
  return (
    <li
      className={cn("relative flex flex-col justify-center px-3 py-1.5 font-mono text-xs leading-4 transition-opacity duration-150 ease-standard", dim && "opacity-35")}
      style={{ height }}
      data-route={route.id}
    >
      <Handle id={`in-${route.id}`} type="target" position={Position.Left} className={wireHandle} style={{ top: "50%" }} />
      {ownLine && name}
      {ownLine && address}
      {!ownLine && (
        <span className="flex gap-3">
          {name}
          {address}
        </span>
      )}
      <Handle id={route.id} type="source" position={Position.Right} className={wireHandle} style={{ top: "50%" }} />
    </li>
  );
};

// The hub every route passes through: one row per exposure, a wire in from its hostname pill on the left and a wire
// out to its service on the right. Row heights come from geometry.ts, so each row is level with its pill.
export const GatewayNode = ({ data, selected }: NodeProps<GatewayNodeType>) => (
  <div className={nodeShell} style={{ width: GATEWAY_W }} data-selected={selected}>
    <div className="flex items-start gap-2.5 px-3 pt-2.5" style={{ height: gatewayHeaderHeight(data) }}>
      {/* Centred on the two lines beside it: the kind over the container's name. */}
      <span className="mt-1 flex size-7 shrink-0 items-center justify-center rounded-md bg-well-shade text-muted-foreground">
        <ShieldIcon className="size-3.5" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-3">
          <span className="min-w-0 flex-1 text-sm leading-5 font-medium">{kindLabels[data.kind]}</span>
          <span className="shrink-0 text-xs text-muted-foreground">
            <span className={cn("relative -top-px mr-2 inline-block size-1.5 rounded-full align-middle", statusDot[data.status])} aria-hidden />
            {data.status}
          </span>
        </div>
        <span className="block font-mono text-xs leading-4 break-words text-muted-foreground">{data.name}</span>
      </div>
    </div>
    <ul className="divide-y divide-border border-t border-border">
      {data.routes.map((r, i) => (
        <RouteItem key={r.id} route={r} height={data.rowHeights?.[i]} dim={!!data.litRoutes && !data.litRoutes.includes(r.id)} />
      ))}
    </ul>
    <div className="border-t border-border px-3 py-2 font-mono text-[11px] leading-4 text-muted-foreground">
      {footerOnOneLine(data) && footerFacts(data).join(" · ")}
      {!footerOnOneLine(data) &&
        footerFacts(data).map((fact) => (
          <span key={fact} className="block break-words">
            {fact}
          </span>
        ))}
    </div>
  </div>
);
