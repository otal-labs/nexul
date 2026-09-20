import { BaseEdge, getBezierPath, type EdgeProps } from "@xyflow/react";

// A route wire: pill → gateway row, or gateway row → service. The row it touches carries the words, so it has none.
export const RouteEdge = (props: EdgeProps) => {
  const [path] = getBezierPath(props);
  return <BaseEdge path={path} style={{ strokeWidth: 1.5, stroke: "var(--muted-foreground)" }} />;
};
