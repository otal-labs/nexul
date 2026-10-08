import { useState } from "react";

import { microheaderClass } from "@/components/Microheader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { TrailDetail } from "@/components/play/TrailDetail";
import { TrailRow } from "@/components/play/TrailRow";
import { useFetchTrails } from "@/hooks/TrailHooks";
import type { PlayType } from "@/models/Play";
import { cn } from "@/lib/utils";

interface TrailSectionProps {
  workspaceId: string;
  targetType: PlayType;
  targetId: string;
  /** Said in place of the list once the trail loads empty. */
  emptyMessage?: string;
  /** Passed to each row; "stacked" suits a narrow column. */
  rowLayout?: "inline" | "stacked";
  className?: string;
}

// Renders nothing with zero trails unless given emptyMessage, so a doc nobody has run a play on shows no empty section.
export const TrailSection = ({ workspaceId, targetType, targetId, emptyMessage, rowLayout = "inline", className }: TrailSectionProps) => {
  const { data: trails, error } = useFetchTrails(targetType, targetId);
  const [openTrailId, setOpenTrailId] = useState<string | null>(null);

  const loadedEmpty = trails && trails.length === 0;
  if (!error && !trails) return null;
  if (!error && loadedEmpty && !emptyMessage) return null;

  return (
    <section className={cn("space-y-0.5 border-t border-border pt-6", className)}>
      <h2 className={cn(microheaderClass, "px-2 pb-1")}>Trail</h2>
      {error && <ErrorDisplay error={error} title="Failed to load the trail" />}
      {loadedEmpty && <p className="px-2 text-xs text-muted-foreground">{emptyMessage}</p>}
      {trails && trails.length > 0 && (
        <ul className="flex flex-col">
          {trails.map((trail) => (
            <TrailRow key={trail.id} workspaceId={workspaceId} trail={trail} onOpen={setOpenTrailId} layout={rowLayout} />
          ))}
        </ul>
      )}
      <TrailDetail trailId={openTrailId} onClose={() => setOpenTrailId(null)} />
    </section>
  );
};
