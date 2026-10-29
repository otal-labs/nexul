import { Handle, Position, type NodeProps } from "@xyflow/react";
import { BoxIcon, DatabaseIcon, GlobeIcon, NetworkIcon, ServerIcon, ShieldIcon, type LucideIcon } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { EXTERNAL_W } from "@/components/topology/geometry";
import { handleClasses, nodeShell } from "@/components/topology/canvasStyles";
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

// Something Nexul does not run; the microheader keeps it from reading as a deployed service.
export const ExternalNode = ({ data, selected }: NodeProps<ExternalNodeType>) => {
  const Icon = labelIcons[data.label] ?? BoxIcon;
  return (
    <div className="group" style={{ width: EXTERNAL_W }}>
      <span className={cn("mb-1.5 block text-[10px] leading-4", microheaderClass)}>outside Nexul</span>
      <div className={cn(nodeShell, "p-3")} data-selected={selected}>
        <Handle type="target" position={Position.Left} className={handleClasses} />
        <div className="flex items-start gap-2">
          <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
          <div className="min-w-0">
            <span className="block text-sm leading-5 font-medium break-words text-pretty">{data.name}</span>
            {data.url && (
              <a className="block font-mono text-xs break-words text-muted-foreground" href={data.url}>
                {data.url.split(/(?<=[/.])/).map((part, i) => (
                  <span key={i}>
                    {part}
                    <wbr />
                  </span>
                ))}
              </a>
            )}
            <span className="font-mono text-[11px] leading-4 text-muted-foreground">{data.label}</span>
          </div>
        </div>
        <Handle type="source" position={Position.Right} className={handleClasses} />
      </div>
    </div>
  );
};
