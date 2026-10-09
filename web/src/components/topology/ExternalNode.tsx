import { Handle, Position, type NodeProps } from "@xyflow/react";
import { GlobeIcon, NetworkIcon, ServerIcon, ShieldIcon, DatabaseIcon, BoxIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { nodeShell } from "@/components/topology/ServiceNode";
import { cn } from "@/lib/utils";
import { ExternalLabel, type ExternalNode as ExternalNodeType } from "@/models/Topology";

const labelIcons: Record<ExternalLabel, LucideIcon> = {
  [ExternalLabel.Domain]: GlobeIcon,
  [ExternalLabel.Tunnel]: ShieldIcon,
  [ExternalLabel.Proxy]: NetworkIcon,
  [ExternalLabel.Database]: DatabaseIcon,
  [ExternalLabel.Api]: ServerIcon,
  [ExternalLabel.Other]: BoxIcon,
};

// Free-form map element for things Nexul does not manage: domains, tunnels, third-party APIs.
export const ExternalNode = ({ data, selected }: NodeProps<ExternalNodeType>) => {
  const Icon = labelIcons[data.label] ?? BoxIcon;
  return (
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
        <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <span className="whitespace-nowrap text-sm font-medium">{data.name}</span>
      </div>
      {data.url && (
        <a className="block whitespace-nowrap font-mono text-xs text-muted-foreground" href={data.url}>
          {data.url}
        </a>
      )}
      <span className="font-mono text-xs text-muted-foreground">{data.label}</span>
      <Handle
        type="source"
        position={Position.Right}
        className="!bg-muted-foreground !transition-transform !duration-150 !ease-standard group-hover:!scale-125"
      />
    </div>
  );
};
