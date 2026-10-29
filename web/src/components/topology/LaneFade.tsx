import { useStore, useViewport } from "@xyflow/react";

import { drawingBelow, drawingLeft, laneBeyond, type Lane } from "@/utils/TopologyLayout";

const fade = "pointer-events-none absolute z-[5] from-surface-2 to-transparent";

// A fade over each field edge the drawing continues past, so a cut reads as "more this way".
export const LaneFade = ({ lanes }: { lanes: Lane[] }) => {
  const viewport = useViewport();
  const width = useStore((s) => s.width);
  const height = useStore((s) => s.height);
  return (
    <>
      {drawingLeft(lanes, viewport) && <div className={`${fade} inset-y-0 left-0 w-16 bg-linear-to-r`} aria-hidden />}
      {laneBeyond(lanes, viewport, width) && <div className={`${fade} inset-y-0 right-0 w-16 bg-linear-to-l`} aria-hidden />}
      {drawingBelow(lanes, viewport, height) && <div className={`${fade} inset-x-0 bottom-0 h-16 bg-linear-to-t`} aria-hidden />}
    </>
  );
};
