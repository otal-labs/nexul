import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { TrailDetail } from "@/components/play/TrailDetail";
import { TrailRow } from "@/components/play/TrailRow";
import { useFetchTrails } from "@/hooks/TrailHooks";
import type { PlayType } from "@/models/Play";

interface TrailSectionProps {
  workspaceId: string;
  targetType: PlayType;
  targetId: string;
  /** Said in place of the list once the trail loads empty; a tab must never open blank. */
  emptyMessage?: string;
}

const microheaderClass =
  "px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Renders nothing with zero trails unless given emptyMessage, so a doc nobody has run a play on shows no empty section.
export const TrailSection = ({ workspaceId, targetType, targetId, emptyMessage }: TrailSectionProps) => {
  const { data: trails, error } = useFetchTrails(targetType, targetId);
  const [openTrailId, setOpenTrailId] = useState<string | null>(null);

  const loadedEmpty = trails && trails.length === 0;
  if (!error && !trails) return null;
  if (!error && loadedEmpty && !emptyMessage) return null;

  return (
    <section className="space-y-0.5 border-t border-border pt-6">
      <h2 className={microheaderClass}>Trail</h2>
      {error && <ErrorDisplay error={error} title="Failed to load the trail" />}
      {loadedEmpty && <EmptyRow>{emptyMessage}</EmptyRow>}
      {trails && trails.length > 0 && (
        <ul className="flex flex-col">
          {trails.map((trail) => (
            <TrailRow key={trail.id} workspaceId={workspaceId} trail={trail} onOpen={setOpenTrailId} />
          ))}
        </ul>
      )}
      <TrailDetail trailId={openTrailId} onClose={() => setOpenTrailId(null)} />
    </section>
  );
};
