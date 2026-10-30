import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Microheader } from "@/components/Microheader";
import { PlayRunList } from "@/components/play/PlayRunList";
import { TrailDetail } from "@/components/play/TrailDetail";
import { useFetchPlayRuns } from "@/hooks/TrailHooks";
import type { Play } from "@/models/Play";

const WAITING_SHOWN = 5;
const RECENT_PAGE = 10;
// ponytail: the list grows its one page up to the server's 100-row cap; reading past it needs offset pages.
const RECENT_MAX = 100;

interface PlayRunsSectionProps {
  play: Play;
}

// The runs waiting on an answer, oldest first and five at a time, over every run newest first; a waiting run is in both.
export const PlayRunsSection = ({ play }: PlayRunsSectionProps) => {
  const [waitingLimit, setWaitingLimit] = useState(WAITING_SHOWN);
  const [recentLimit, setRecentLimit] = useState(RECENT_PAGE);
  const [openTrailId, setOpenTrailId] = useState<string | null>(null);
  const { data: waiting } = useFetchPlayRuns(play.id, true, waitingLimit);
  const { data: recent, error } = useFetchPlayRuns(play.id, false, recentLimit);

  return (
    <section aria-labelledby="play-runs" className="@container space-y-3">
      <div className="flex h-7 items-center gap-2">
        <Microheader id="play-runs">Runs</Microheader>
        {recent && recent.total > 0 && <span className="font-mono text-[11px] text-muted-foreground tabular-nums">{recent.total}</span>}
      </div>
      {error && <ErrorDisplay error={error} title="Couldn't load the runs." />}
      {waiting && waiting.total > 0 && (
        <PlayRunList label="Needs answer" count={waiting.total} runs={waiting.items} onOpen={setOpenTrailId}>
          {waiting.total > waiting.items.length && (
            <Button variant="ghost" size="sm" onClick={() => setWaitingLimit(Math.min(waiting.total, RECENT_MAX))}>
              Show all {waiting.total}
            </Button>
          )}
        </PlayRunList>
      )}
      {recent && recent.total === 0 && (
        <div className="rounded-lg border bg-card">
          <EmptyRow>No runs yet</EmptyRow>
        </div>
      )}
      {recent && recent.total > 0 && (
        <PlayRunList label="Recent" runs={recent.items} onOpen={setOpenTrailId}>
          {recent.total > recent.items.length && recentLimit < RECENT_MAX && (
            <Button variant="ghost" size="sm" onClick={() => setRecentLimit(Math.min(recentLimit + RECENT_PAGE, RECENT_MAX))}>
              Show more
            </Button>
          )}
        </PlayRunList>
      )}
      <TrailDetail trailId={openTrailId} onClose={() => setOpenTrailId(null)} />
    </section>
  );
};
