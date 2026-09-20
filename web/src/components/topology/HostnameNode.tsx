import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ExternalLinkIcon, GlobeIcon } from "lucide-react";

import type { HostnameNode as HostnameNodeType } from "@/models/Topology";

// A public entry point, derived from an exposure. Sized to its text: nothing on the canvas truncates.
export const HostnameNode = ({ data }: NodeProps<HostnameNodeType>) => (
  <div className="flex h-10 w-max items-center gap-2 rounded-full border border-border bg-card px-3 shadow-card">
    <GlobeIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    <a
      href={`https://${data.hostname}`}
      target="_blank"
      rel="noreferrer"
      className="nodrag whitespace-nowrap font-mono text-xs underline-offset-4 hover:underline"
    >
      {data.hostname}
    </a>
    <ExternalLinkIcon className="size-3 shrink-0 text-muted-foreground" aria-hidden />
    <Handle type="source" position={Position.Right} className="!bg-muted-foreground" />
  </div>
);
