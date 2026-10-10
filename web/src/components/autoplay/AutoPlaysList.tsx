import { PlusIcon } from "lucide-react";

import { AutoPlayCapLine } from "@/components/autoplay/AutoPlayCapLine";
import { AutoPlayRow } from "@/components/autoplay/AutoPlayRow";
import { PlayQueueNowLine } from "@/components/autoplay/PlayQueueNowLine";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchAutoPlays } from "@/hooks/PlayHooks";
import { useRowGlide } from "@/hooks/useRowGlide";
import type { AutoPlay, AutoPlaySubject } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

interface AutoPlaysListProps {
  play: Play;
  onOpen: (autoPlay: AutoPlay | null) => void;
}

export const AutoPlaysList = ({ play, onOpen }: AutoPlaysListProps) => {
  const subject: AutoPlaySubject = play.type === "doc" ? "doc" : "ticket";
  const can = useAreaAccess();
  const canWrite = can?.("editAutoPlays") ?? false;
  const canDelete = can?.("deleteAutoPlays") ?? false;
  const { data: autoPlays, error, isPending } = useFetchAutoPlays(play.workspace_id, play.id);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  return (
    <div className="space-y-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {autoPlays && autoPlays.length === 0 && (
        <EmptyRow flush>No auto plays yet. Add one and this play starts by itself when a {subject} matches it.</EmptyRow>
      )}
      {autoPlays && autoPlays.length > 0 && (
        <div ref={glideRef}>
          <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
            {autoPlays.map((autoPlay) => (
              <AutoPlayRow
                key={autoPlay.id}
                autoPlay={autoPlay}
                subject={subject}
                canWrite={canWrite}
                canDelete={canDelete}
                onOpen={onOpen}
                onLeave={prepareGlide}
              />
            ))}
          </EnterList>
        </div>
      )}
      {canWrite && (
        <Button type="button" variant="outline" size="sm" onClick={() => onOpen(null)}>
          <PlusIcon />
          Add auto play
        </Button>
      )}
      <div className="space-y-1 empty:hidden">
        <PlayQueueNowLine play={play} />
        {subject === "ticket" && <AutoPlayCapLine workspaceId={play.workspace_id} canWrite={canWrite} />}
      </div>
    </div>
  );
};
