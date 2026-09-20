import { ChevronDown, LoaderCircle, Sparkles, Square } from "lucide-react";
import { useState } from "react";

import { PlayMenuRow } from "@/components/play/PlayMenuRow";
import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useHarnessReadiness } from "@/hooks/PairingHooks";
import { useFetchApplicablePlays } from "@/hooks/PlayHooks";
import { useActiveTrail, useStopTrail } from "@/hooks/TrailHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { HarnessReadiness } from "@/models/Pairing";
import type { Play } from "@/models/Play";

const disabledReason = (running: boolean, readiness: HarnessReadiness | undefined): string => {
  if (running) return "a run is in progress";
  if (readiness && readiness.state !== "ready") return readiness.message;
  return "";
};

interface PlaysMenuProps {
  workspaceId: string;
  projectId: string;
  docId: string;
}

// Hidden without plays:run, without docs:thread, or with no doc play applicable; the starter or a
// plays:write holder gets a Stop row for the active trail instead of the disabled play list.
export const PlaysMenu = ({ workspaceId, projectId, docId }: PlaysMenuProps) => {
  const canRun = useHasPermission("plays:run");
  const canThread = useHasPermission("docs:thread");
  const canWrite = useHasPermission("plays:write");
  const { data: me } = useFetchMe();
  const { data: plays } = useFetchApplicablePlays(workspaceId, projectId, "doc", undefined);
  const readiness = useHarnessReadiness(projectId);
  const activeTrail = useActiveTrail("doc", docId);
  const stopTrail = useStopTrail();
  const [menuOpen, setMenuOpen] = useState(false);
  const [chosenPlay, setChosenPlay] = useState<Play | null>(null);

  if (!canRun || !canThread) return null;
  if (!plays || plays.length === 0) return null;

  const running = activeTrail !== undefined;
  const canStop = running && (activeTrail.starter_id === me?.user.id || canWrite);
  const reason = disabledReason(running, readiness);
  const rowsDisabled = reason !== "" || readiness === undefined;

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
          {reason !== "" && <p className="px-2 py-1.5 font-mono text-[11px] text-muted-foreground">{reason}</p>}
          {canStop && (
            <Button
              variant="outline"
              size="sm"
              className="mb-1 w-full border-warning/50"
              disabled={stopTrail.isPending}
              onClick={() => stopTrail.mutate(activeTrail.id)}
            >
              <LoaderCircle className="size-3.5 animate-spin text-warning" aria-hidden />
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
