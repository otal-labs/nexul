import { useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { TrailDetail } from "@/components/play/TrailDetail";
import { TrailRow } from "@/components/play/TrailRow";
import { useFetchTrails } from "@/hooks/TrailHooks";
import type { PlayType } from "@/models/Play";

interface TrailSectionProps {
  workspaceId: string;
  targetType: PlayType;
  targetId: string;
}

const microheaderClass =
  "px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Renders nothing with zero trails, so a ticket nobody has run a play on shows no empty section.
export const TrailSection = ({ workspaceId, targetType, targetId }: TrailSectionProps) => {
  const { data: trails, error } = useFetchTrails(targetType, targetId);
  const [openTrailId, setOpenTrailId] = useState<string | null>(null);

  if (!error && (!trails || trails.length === 0)) return null;

  return (
    <section className="space-y-0.5 border-t border-border pt-6">
      <h2 className={microheaderClass}>Trail</h2>
      {error && <ErrorDisplay error={error} title="Failed to load the trail" />}
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
