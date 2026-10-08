import type { NetworkRect } from "@/utils/TopologyLayout";

interface NetworkBoxProps {
  rect: NetworkRect;
}

// Mounted via ViewportPortal so pan/zoom move it with the nodes; keyed by network name so it never replays on pan.
export const NetworkBox = ({ rect }: NetworkBoxProps) => (
  <div
    className="animate-in fade-in-0 absolute rounded-lg border border-dashed border-border duration-200 ease-out"
    style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height, zIndex: -1, pointerEvents: "none" }}
    data-network={rect.name}
  >
    <span className="absolute left-3 top-2 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
      network · {rect.name}
    </span>
  </div>
);
