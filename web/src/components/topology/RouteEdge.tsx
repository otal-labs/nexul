import { BaseEdge, getBezierPath, type EdgeProps } from "@xyflow/react";

import { wireStroke } from "@/components/topology/canvasStyles";
import type { RelationEdgeData } from "@/models/Topology";

// The row a route wire touches carries the words, so the wire has none.
export const RouteEdge = (props: EdgeProps) => {
  const [path] = getBezierPath(props);
  const tone = (props.data as RelationEdgeData | undefined)?.tone;
  return <BaseEdge path={path} style={{ strokeWidth: 1.5, stroke: wireStroke(tone) }} />;
};
