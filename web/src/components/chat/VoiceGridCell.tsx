import type { ReactNode } from "react";

interface VoiceGridCellProps {
  children: ReactNode;
}

// Centres a 16:9 box in the grid cell, as large as the cell's height or width allows, so a lone tile never becomes a wide strip.
export const VoiceGridCell = ({ children }: VoiceGridCellProps) => (
  <div className="flex items-center justify-center [container-type:size]">
    <div className="aspect-video w-[min(100%,calc(100cqh*16/9))] *:size-full">{children}</div>
  </div>
);
