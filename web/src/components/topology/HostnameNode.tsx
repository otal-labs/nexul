import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ExternalLinkIcon, GlobeIcon } from "lucide-react";

import { hostnameLines, PILL_MAX_W } from "@/components/topology/geometry";
import { statusDot, wireHandle } from "@/components/topology/canvasStyles";
import { cn } from "@/lib/utils";
import type { HostnameNode as HostnameNodeType } from "@/models/Topology";

// A 12px radius is a full pill at one line and keeps those ends when the hostname wraps into geometry.ts's lines.
export const HostnameNode = ({ data }: NodeProps<HostnameNodeType>) => (
  <div className="canvas-card flex min-h-6 w-max items-start gap-2 rounded-xl px-3 py-[3px]" style={{ maxWidth: PILL_MAX_W }}>
    {data.tone && <span className={cn("mt-[5px] size-1.5 shrink-0 rounded-full", statusDot[data.tone])} aria-label={data.tone} />}
    <GlobeIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    <a href={`https://${data.hostname}`} target="_blank" rel="noreferrer" className="nodrag min-w-0 font-mono text-xs leading-4 underline-offset-4 hover:underline">
      {hostnameLines(data).map((line) => (
        <span key={line} className="block whitespace-nowrap">
          {line}
        </span>
      ))}
    </a>
    <ExternalLinkIcon className="mt-0.5 size-3 shrink-0 text-muted-foreground" aria-hidden />
    <Handle type="source" position={Position.Right} className={wireHandle} />
  </div>
);
