import { useViewport, useViewportStore } from "@xyflow/react";

import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { headerTop, type Lane } from "@/utils/TopologyLayout";

// Column names on one line above the lanes in view, so a newcomer reads the map's grammar without a legend.
export const LaneHeaders = ({ lanes }: { lanes: Lane[] }) => {
  const viewport = useViewport();
  const width = useViewportStore((s) => s.width);
  const top = headerTop(lanes, viewport, width);
  return lanes
    .filter((l) => l.label)
    .map((l) => (
      <span key={l.label} className={cn("absolute whitespace-nowrap", microheaderClass)} style={{ left: l.left, top, pointerEvents: "none" }}>
        {l.label}
      </span>
    ));
};
