import { TrackRefContext, useVisualStableUpdate } from "@livekit/components-react";
import type { TrackReferenceOrPlaceholder } from "@livekit/components-core";

import { VoicePeoplePopover } from "@/components/voice/VoicePeoplePopover";
import { VoiceTile } from "@/components/voice/VoiceTile";
import { useElementSize } from "@/hooks/useElementSize";
import { stageGrid } from "@/lib/stageGrid";

interface VoiceStageGridProps {
  tracks: TrackReferenceOrPlaceholder[];
}

const trackKey = (track: TrackReferenceOrPlaceholder) => `${track.participant.identity}:${track.source}`;

// Tiles sized to the measured tile area on both axes; a last row that is not full centres, and the rest go behind +N.
export const VoiceStageGrid = ({ tracks }: VoiceStageGridProps) => {
  const [ref, { width, height }] = useElementSize<HTMLDivElement>();
  const grid = stageGrid(tracks.length, width, height);
  const size = { width: Math.floor(grid.cellWidth), height: Math.floor(grid.cellHeight) };
  // Keeps the tiles where they are and swaps whoever starts talking out from behind the +N.
  const ordered = useVisualStableUpdate(tracks, grid.shown);
  return (
    <div ref={ref} className="min-h-0 flex-1">
      {width > 0 && (
        <div className="flex size-full flex-wrap content-center justify-center gap-3">
          {ordered.slice(0, grid.shown).map((track) => (
            <TrackRefContext.Provider key={trackKey(track)} value={track}>
              <VoiceTile style={size} />
            </TrackRefContext.Provider>
          ))}
          {grid.overflow > 0 && (
            <VoicePeoplePopover>
              <button
                type="button"
                style={size}
                className="flex shrink-0 items-center justify-center rounded-md border border-border bg-card font-mono text-lg text-muted-foreground tabular-nums hover:bg-accent"
                aria-label={`${grid.overflow} more in the call`}
              >
                +{grid.overflow}
              </button>
            </VoicePeoplePopover>
          )}
        </div>
      )}
    </div>
  );
};
