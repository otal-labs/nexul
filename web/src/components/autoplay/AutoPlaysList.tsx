import { PlusIcon } from "lucide-react";

import { AutoPlayCapLine } from "@/components/autoplay/AutoPlayCapLine";
import { AutoPlayRow } from "@/components/autoplay/AutoPlayRow";
import { ComputerRunsLine } from "@/components/autoplay/ComputerRunsLine";
import { PlayQueueNowLine } from "@/components/autoplay/PlayQueueNowLine";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { ActionButton } from "@/components/action/ActionButton";
import { useActionState } from "@/hooks/useActionState";
import { useFetchAutoPlays } from "@/hooks/PlayHooks";
import { useRowGlide } from "@/hooks/useRowGlide";
import { subjectOf, type AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

interface AutoPlaysListProps {
  play: Play;
  onOpen: (autoPlay: AutoPlay | null) => void;
}

export const AutoPlaysList = ({ play, onOpen }: AutoPlaysListProps) => {
  const subject = subjectOf(play);
  const canWrite = useActionState({ permission: "autoplays:write" }).state === "live";
  const { data: autoPlays, error, isPending } = useFetchAutoPlays(play.workspace_id, play.id);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  return (
    <section aria-labelledby="play-auto-plays" className="space-y-3">
      <div className="flex h-7 items-center gap-2">
        <Microheader id="play-auto-plays">Starts by itself</Microheader>
        {autoPlays && autoPlays.length > 0 && <span className="font-mono text-[11px] text-muted-foreground tabular-nums">{autoPlays.length}</span>}
        <ActionButton permission="autoplays:write" verb="add auto plays" icon={PlusIcon} type="button" variant="ghost" size="sm" className="ml-auto" aria-label="Add auto play" onClick={() => onOpen(null)}>
          Add
        </ActionButton>
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {autoPlays && (
        <div ref={glideRef} className="overflow-hidden rounded-lg border bg-card">
          {autoPlays.length === 0 && <EmptyRow className="text-pretty">Starts only when someone runs it. Add an auto play and it starts by itself when a {subject} matches.</EmptyRow>}
          {autoPlays.length > 0 && (
            <EnterList className="divide-y divide-border">
              {autoPlays.map((autoPlay) => (
                <AutoPlayRow key={autoPlay.id} autoPlay={autoPlay} subject={subject} canWrite={canWrite} onOpen={onOpen} onLeave={prepareGlide} />
              ))}
            </EnterList>
          )}
          <div className="space-y-1 border-t bg-surface-2 px-4 py-2.5 empty:hidden">
            <PlayQueueNowLine play={play} />
            {subject === "ticket" && <AutoPlayCapLine workspaceId={play.workspace_id} canWrite={canWrite} />}
            <ComputerRunsLine />
          </div>
        </div>
      )}
    </section>
  );
};
