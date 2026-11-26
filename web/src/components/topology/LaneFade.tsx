import { useReactFlow, useViewport, useViewportStore } from "@xyflow/react";
import { ArrowLeftIcon, ArrowRightIcon } from "lucide-react";

import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { drawingBelow, headerTop, laneBeyond, laneBefore, panBackTo, panTo, type Lane } from "@/utils/TopologyLayout";

const fade = "pointer-events-none absolute z-[5] from-surface-2 to-transparent";
const cue = cn(
  microheaderClass,
  "absolute z-[6] flex cursor-pointer items-center gap-1.5 rounded-sm bg-surface-2 px-1.5 py-1 transition-colors duration-150 ease-standard hover:text-foreground [&_svg]:size-3",
);

// A fade over each field edge the drawing continues past, so a cut reads as "more this way"; on the header row the
// next lane's name continues the column heads at the edge and pans there.
export const LaneFade = ({ lanes }: { lanes: Lane[] }) => {
  const { setViewport } = useReactFlow();
  const viewport = useViewport();
  const width = useViewportStore((s) => s.width);
  const height = useViewportStore((s) => s.height);
  const before = laneBefore(lanes, viewport);
  const beyond = laneBeyond(lanes, viewport, width);
  const headerY = lanes.length > 0 ? viewport.y + headerTop(lanes, viewport, width) * viewport.zoom - 4 : 0;
  const top = Math.min(Math.max(headerY, 8), Math.max(height - 32, 8));
  return (
    <>
      {before && <div className={`${fade} inset-y-0 left-0 w-24 bg-linear-to-r`} aria-hidden />}
      {beyond && <div className={`${fade} inset-y-0 right-0 w-24 bg-linear-to-l`} aria-hidden />}
      {drawingBelow(lanes, viewport, height) && <div className={`${fade} inset-x-0 bottom-0 h-16 bg-linear-to-t`} aria-hidden />}
      {before && (
        <button type="button" className={cn(cue, "left-2")} style={{ top }} onClick={() => void setViewport(panBackTo(before, viewport))}>
          <ArrowLeftIcon aria-hidden />
          {before.label || "Back"}
        </button>
      )}
      {beyond && (
        <button type="button" className={cn(cue, "right-2")} style={{ top }} onClick={() => void setViewport(panTo(beyond, viewport, width))}>
          {beyond.label || "More"}
          <ArrowRightIcon aria-hidden />
        </button>
      )}
    </>
  );
};
