import { useState } from "react";
import { BaseEdge, EdgeLabelRenderer, getSmoothStepPath, type EdgeProps } from "@xyflow/react";

import { cn } from "@/lib/utils";
import { wireStroke } from "@/components/topology/canvasStyles";
import { RelationKind, type AttentionStatus } from "@/models/Topology";

const kindLabels: Record<RelationKind, string> = {
  [RelationKind.DependsOn]: "depends on",
  [RelationKind.ConnectsTo]: "connects to",
  [RelationKind.Mounts]: "mounts",
};

// One stroke per kind, so the three read apart before their labels do (the arrow comes from markerEnd).
const kindDash: Record<RelationKind, string | undefined> = {
  [RelationKind.DependsOn]: "4 4",
  [RelationKind.ConnectsTo]: "1 4",
  [RelationKind.Mounts]: undefined,
};

// Past a network box's 24px padding, so a relation looping between two cards in one box never runs along its edge.
const LOOP_OFFSET = 36;

// On a straight stretch, never the step: beside a loop, by the target, or by the source for a port (its target's boxed).
const labelAt = (p: EdgeProps, midX: number, midY: number, port: boolean) => {
  if (p.sourcePosition === p.targetPosition) return `translate(6px, -50%) translate(${midX}px, ${midY}px)`;
  if (port) return `translate(8px, -50%) translate(${p.sourceX}px, ${p.sourceY}px)`;
  return `translate(calc(-100% - 10px), -50%) translate(${p.targetX}px, ${p.targetY}px)`;
};

// Labels sit in the label layer above every wire; hover is a style swap, as a flow animation repaints every edge.
export const RelationEdge = (props: EdgeProps) => {
  const [path, labelX, labelY] = getSmoothStepPath({ ...props, offset: LOOP_OFFSET, borderRadius: 8 });
  const kind = props.data?.kind as RelationKind | undefined;
  const override = props.data?.label as string | undefined;
  const label = override ?? (kind ? (kindLabels[kind] ?? kind) : null);
  const [hovered, setHovered] = useState(false);

  const emphasized = props.selected || hovered;

  return (
    <g onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}>
      <BaseEdge
        path={path}
        {...(props.markerEnd ? { markerEnd: props.markerEnd } : {})}
        style={{
          strokeDasharray: kind ? kindDash[kind] : "4 4",
          strokeLinecap: "round",
          strokeWidth: emphasized ? 2 : 1.5,
          stroke: wireStroke(props.data?.tone as AttentionStatus | undefined),
          transition: "stroke 150ms var(--ease-standard), stroke-width 150ms var(--ease-standard)",
          ...(hovered && !props.selected ? { stroke: "var(--ring)" } : {}),
        }}
      />
      {label && (
        <EdgeLabelRenderer>
          <span
            className={cn(
              "absolute rounded-sm bg-surface-2 px-1 font-mono text-[10px] leading-4 text-muted-foreground transition-opacity duration-150 ease-standard",
              emphasized && "text-primary outline-1 outline-ring",
              props.data?.dim === true && "opacity-35",
            )}
            style={{ transform: labelAt(props, labelX, labelY, !!override) }}
          >
            {label}
          </span>
        </EdgeLabelRenderer>
      )}
    </g>
  );
};
