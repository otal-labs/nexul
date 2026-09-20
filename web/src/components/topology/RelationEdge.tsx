import { useState } from "react";
import { BaseEdge, getSmoothStepPath, type EdgeProps } from "@xyflow/react";

import { cn } from "@/lib/utils";
import { RelationKind } from "@/models/Topology";

const kindLabels: Record<RelationKind, string> = {
  [RelationKind.DependsOn]: "depends on",
  [RelationKind.ConnectsTo]: "connects to",
  [RelationKind.Mounts]: "mounts",
};

// Mono glyph advance at 10px, keeps the pill snug without measuring text in SVG.
const GLYPH = 6;
const PILL_H = 16;
const PILL_RADIUS = 8;

// Hover feedback is a plain style swap, not a "flow" animation, since that would repaint every edge every frame.
export const RelationEdge = (props: EdgeProps) => {
  const [path, labelX, labelY] = getSmoothStepPath(props);
  const kind = props.data?.kind as RelationKind | undefined;
  const override = props.data?.label as string | undefined;
  const label = override ?? (kind ? (kindLabels[kind] ?? kind) : null);
  const [hovered, setHovered] = useState(false);

  const pillY = labelY - PILL_H - 4;
  const pillW = label ? label.length * GLYPH + 16 : 0;
  const emphasized = props.selected || hovered;

  return (
    <g onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}>
      <BaseEdge
        path={path}
        style={{
          strokeDasharray: "4 4",
          strokeWidth: emphasized ? 2 : 1.5,
          transition: "stroke 150ms var(--ease-standard), stroke-width 150ms var(--ease-standard)",
          ...(hovered && !props.selected ? { stroke: "var(--ring)" } : {}),
        }}
      />
      {label && (
        <g transform={`translate(${labelX - pillW / 2}, ${pillY})`}>
          <rect
            width={pillW}
            height={PILL_H}
            rx={PILL_RADIUS}
            className={cn("fill-surface-2 stroke-border stroke-1", emphasized && "stroke-ring")}
          />
          <text
            x={pillW / 2}
            y={PILL_H / 2 + 3.5}
            textAnchor="middle"
            className={cn("fill-muted-foreground font-mono text-[10px]", emphasized && "fill-primary")}
          >
            {label}
          </text>
        </g>
      )}
    </g>
  );
};
