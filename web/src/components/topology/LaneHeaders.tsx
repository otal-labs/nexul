import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import type { Lane } from "@/utils/TopologyLayout";

// Column names on one line above the drawing, so a newcomer reads the map's grammar without a legend.
export const LaneHeaders = ({ lanes }: { lanes: Lane[] }) => {
  const top = Math.min(...lanes.map((l) => l.top)) - 28;
  return lanes
    .filter((l) => l.label)
    .map((l) => (
      <span key={l.label} className={cn("absolute whitespace-nowrap", microheaderClass)} style={{ left: l.left, top, pointerEvents: "none" }}>
        {l.label}
      </span>
    ));
};
