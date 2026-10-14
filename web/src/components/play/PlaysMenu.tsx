import { ChevronDown, LoaderCircle, Sparkles, Square } from "lucide-react";
import { useState } from "react";

import { FixLink } from "@/components/FixLink";
import { PlayMenuRow } from "@/components/play/PlayMenuRow";
import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchApplicablePlays } from "@/hooks/PlayHooks";
import { useActiveTrail, useStopTrail } from "@/hooks/TrailHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useRunBlock } from "@/hooks/PairingProjectHooks";
import type { Play } from "@/models/Play";

const disabledReason = (running: boolean, blocked: string | undefined): string => {
  if (running) return "a run is in progress";
  return blocked ?? "";
};

interface PlaysMenuProps {
  workspaceId: string;
  projectId: string;
  docId: string;
}

// Hidden without docs:thread or with no doc play the viewer can run; the starter or a
// plays:write holder gets a Stop row for the active trail instead of the disabled play list.
export const PlaysMenu = ({ workspaceId, projectId, docId }: PlaysMenuProps) => {
  const canThread = useHasPermission("docs:thread");
  const canWrite = useHasPermission("plays:write");
  const { data: me } = useFetchMe();
  const { data: applicable } = useFetchApplicablePlays(workspaceId, projectId, "doc", undefined);
  const plays = applicable?.filter((p) => p.can_run);
  const block = useRunBlock(projectId);
  const blocked = block?.message;
  const activeTrail = useActiveTrail("doc", docId);
  const stopTrail = useStopTrail();
  const [menuOpen, setMenuOpen] = useState(false);
  const [chosenPlay, setChosenPlay] = useState<Play | null>(null);

  if (!canThread) return null;
  if (!plays || plays.length === 0) return null;

  const running = activeTrail !== undefined;
  const canStop = running && (activeTrail.starter_id === me?.user.id || canWrite);
  const reason = disabledReason(running, blocked);
  const rowsDisabled = reason !== "" || blocked === undefined;

  const choose = (play: Play) => {
    setMenuOpen(false);
    setChosenPlay(play);
  };

  return (
    <>
      <Popover open={menuOpen} onOpenChange={setMenuOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm">
            <Sparkles className="size-3.5" aria-hidden />
            Plays
            <ChevronDown className="size-3.5 text-muted-foreground" aria-hidden />
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-72 p-1">
          {reason !== "" && (
            <p className="px-2 py-1.5 font-mono text-xs text-muted-foreground">
              {reason} {!running && block?.link && <FixLink {...block.link} />}
            </p>
          )}
          {canStop && (
            <Button
              variant="outline"
              size="sm"
              className="mb-1 w-full border-warning/50"
              disabled={stopTrail.isPending}
              onClick={() => stopTrail.mutate(activeTrail.id)}
            >
              <LoaderCircle className="size-3.5 animate-spin motion-reduce:animate-none text-warning" aria-hidden />
              Stop {activeTrail.play_label}
              <Square className="ml-1 size-3 fill-current" aria-hidden />
            </Button>
          )}
          {plays.map((play) => (
            <PlayMenuRow key={play.id} play={play} disabled={rowsDisabled} onChoose={choose} />
          ))}
        </PopoverContent>
      </Popover>
      {chosenPlay && (
        <PlayRunDialog
          play={chosenPlay}
          projectId={projectId}
          targetType="doc"
          targetId={docId}
          open={chosenPlay !== null}
          onClose={() => setChosenPlay(null)}
        />
      )}
    </>
  );
};
