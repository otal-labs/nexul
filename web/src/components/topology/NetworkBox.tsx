import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import type { NetworkRect } from "@/utils/TopologyLayout";

interface NetworkBoxProps {
  rect: NetworkRect;
}

// Mounted via ViewportPortal so pan/zoom move it with the nodes; keyed by network name so it never replays on pan.
// Opaque in the canvas colour with the tint on top, so the dot grid stops at the box's edge.
export const NetworkBox = ({ rect }: NetworkBoxProps) => (
  <div
    className={cn(
      "animate-in fade-in-0 absolute rounded-xl bg-surface-2 ring-1 ring-border transition-opacity duration-150 ease-standard before:absolute before:inset-0 before:rounded-[inherit] before:bg-foreground/[0.035]",
      rect.dim && "opacity-35",
    )}
    style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height, zIndex: -1, pointerEvents: "none" }}
    data-network={rect.name}
  >
    <span className="absolute top-2.5 right-3 left-3 flex flex-col">
      <span className={cn("flex items-start gap-2", microheaderClass)}>
        <span className="min-w-0 break-words">
          network ·{" "}
          <span className="normal-case">
            {rect.name.split(/(?<=[-_])/).map((part, i) => (
              <span key={i}>
                {part}
                <wbr />
              </span>
            ))}
          </span>
        </span>
        <span className="ml-auto shrink-0 tabular-nums">{rect.count}</span>
      </span>
      {rect.machine && <span className="truncate font-mono text-[11px] leading-4 text-muted-foreground">{rect.machine}</span>}
    </span>
  </div>
);
