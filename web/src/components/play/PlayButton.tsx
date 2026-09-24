import { CircleHelp, LoaderCircle, Sparkles, Square } from "lucide-react";
import { useState } from "react";

import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { TrailDetail } from "@/components/play/TrailDetail";
import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useHarnessReadiness } from "@/hooks/PairingHooks";
import { useActiveTrail, useStopTrail } from "@/hooks/TrailHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { HarnessReadiness } from "@/models/Pairing";
import type { Play, PlayType } from "@/models/Play";
import { cn } from "@/lib/utils";

interface PlayButtonProps {
  play: Play;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  variant?: "outline" | "ghost" | "default";
  // Replaces the play's own label on the idle button, as the Interview page's Run and Re-run do.
  label?: string;
  className?: string;
}

const disabledReason = (running: boolean, waiting: boolean, readiness: HarnessReadiness | undefined): string => {
  if (waiting) return "a run is waiting for an answer";
  if (running) return "a run is in progress";
  if (readiness && readiness.state !== "ready") return readiness.message;
  return "";
};

// Hidden without plays:run; the starter or a plays:write holder gets Stop while a run occupies the target.
export const PlayButton = ({ play, projectId, targetType, targetId, variant = "outline", label, className }: PlayButtonProps) => {
  const canRun = useHasPermission("plays:run");
  const canWrite = useHasPermission("plays:write");
  const { data: me } = useFetchMe();
  const readiness = useHarnessReadiness(projectId);
  const activeTrail = useActiveTrail(targetType, targetId);
  const liveState = usePlayRunStore((s) => (activeTrail ? s.frames[activeTrail.id]?.state : undefined));
  const stopTrail = useStopTrail();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [trailOpen, setTrailOpen] = useState(false);

  if (!canRun) return null;

  const running = activeTrail !== undefined;
  const canStop = running && (activeTrail.starter_id === me?.user.id || canWrite);
  // Only the play that asked reads as waiting; its siblings stay disabled with the reason.
  const waiting = running && (liveState ?? activeTrail.state) === "waiting" && activeTrail.play_id === play.id;
  const reason = disabledReason(running, (liveState ?? activeTrail?.state) === "waiting", readiness);

  return (
    <span className={cn("inline-flex flex-col items-start gap-1", className)}>
      {waiting && (
        <span className="inline-flex items-center gap-1">
          <Button
            variant="outline"
            size="sm"
            className="border-info/50"
            title={`${play.label} asked a question`}
            onClick={() => setTrailOpen(true)}
          >
            <CircleHelp className="size-3.5 text-info" aria-hidden />
            Waiting for your answer
          </Button>
          {canStop && (
            <Button
              size="icon"
              variant="ghost"
              className="size-8"
              title="Stop this run"
              aria-label={`Stop ${play.label}`}
              disabled={stopTrail.isPending}
              onClick={() => stopTrail.mutate(activeTrail.id)}
            >
              <Square className="size-3 fill-current" aria-hidden />
            </Button>
          )}
        </span>
      )}
      {!waiting && canStop && (
        <Button
          variant="outline"
          size="sm"
          className="border-warning/50"
          title="Stop this run"
          aria-label={`Stop ${play.label}`}
          disabled={stopTrail.isPending}
          onClick={() => stopTrail.mutate(activeTrail.id)}
        >
          <LoaderCircle className="size-3.5 animate-spin text-warning" aria-hidden />
          {play.label}
          <Square className="ml-1 size-3 fill-current" aria-hidden />
        </Button>
      )}
      {!waiting && !canStop && (
        <Button
          variant={variant}
          size="sm"
          disabled={reason !== "" || readiness === undefined}
          title={reason || play.description}
          onClick={() => setDialogOpen(true)}
        >
          {running && <LoaderCircle className="size-3.5 animate-spin text-warning" aria-hidden />}
          {!running && <Sparkles className="size-3.5" aria-hidden />}
          {label ?? play.label}
        </Button>
      )}
      {!waiting && !canStop && reason !== "" && <span className="font-mono text-[11px] text-muted-foreground">{reason}</span>}
      {activeTrail && <TrailDetail trailId={trailOpen ? activeTrail.id : null} onClose={() => setTrailOpen(false)} />}
      <PlayRunDialog
        play={play}
        projectId={projectId}
        targetType={targetType}
        targetId={targetId}
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    </span>
  );
};
