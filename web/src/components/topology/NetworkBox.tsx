import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import type { NetworkRect } from "@/utils/TopologyLayout";

interface NetworkBoxProps {
  rect: NetworkRect;
}

// Mounted via ViewportPortal so pan/zoom move it with the nodes; keyed by network name so it never replays on pan.
export const NetworkBox = ({ rect }: NetworkBoxProps) => (
  <div
    className="animate-in fade-in-0 absolute rounded-xl bg-foreground/[0.025] ring-1 ring-border duration-200 ease-out"
    style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height, zIndex: -1, pointerEvents: "none" }}
    data-network={rect.name}
  >
    <span className={cn("absolute top-2.5 left-3 whitespace-nowrap", microheaderClass)}>
      network · {rect.name}
    </span>
  </div>
);
