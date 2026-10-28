import { CircleHelp, LoaderCircle, Sparkles, Square } from "lucide-react";
import { useState } from "react";

import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { TrailDetail } from "@/components/play/TrailDetail";
import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useActiveTrail, useStopTrail } from "@/hooks/TrailHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useRunBlockedReason } from "@/hooks/PairingProjectHooks";
import type { ListedPlay, PlayType } from "@/models/Play";
import { cn } from "@/lib/utils";

interface PlayButtonProps {
  play: ListedPlay;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  variant?: "outline" | "ghost" | "default";
  // Replaces the play's own label on the idle button, as the Interview page's Run and Re-run do.
  label?: string;
  // Swaps the idle icon for the warning dot: what the play last ran on has changed since.
  outOfDate?: boolean;
  className?: string;
  // Styles the play's own button, apart from the Stop beside it.
  mainClassName?: string;
}

const disabledReason = (running: boolean, waiting: boolean, blocked: string | undefined): string => {
  if (waiting) return "a run is waiting for an answer";
  if (running) return "a run is in progress";
  return blocked ?? "";
};

// Hidden unless the play's can_run says the viewer may run it; while it runs it opens the run, and the starter or a plays:write holder gets Stop beside it.
export const PlayButton = ({ play, projectId, targetType, targetId, variant = "outline", label, outOfDate = false, className, mainClassName }: PlayButtonProps) => {
  const canWrite = useHasPermission("plays:write");
  const { data: me } = useFetchMe();
  const blocked = useRunBlockedReason(projectId);
  const activeTrail = useActiveTrail(targetType, targetId);
  const liveState = usePlayRunStore((s) => (activeTrail ? s.frames[activeTrail.id]?.state : undefined));
  const stopTrail = useStopTrail();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [trailOpen, setTrailOpen] = useState(false);

  if (!play.can_run) return null;

  const running = activeTrail !== undefined;
  // Only the running play's own button says so; a sibling keeps its icon and gives the reason on hover.
  const ownRun = running && activeTrail.play_id === play.id;
  const canStop = ownRun && (activeTrail.starter_id === me?.user.id || canWrite);
  const waiting = ownRun && (liveState ?? activeTrail.state) === "waiting";
  const anyWaiting = (liveState ?? activeTrail?.state) === "waiting";
  const reason = disabledReason(running, anyWaiting, blocked);

  return (
    // A disabled button takes no pointer events, so the reason's tooltip sits on the wrapper.
    <span className={cn("inline-flex items-center gap-1", className)} title={ownRun ? undefined : reason || play.description || play.label}>
      {ownRun && (
        <Button
          variant={variant === "default" ? "outline" : variant}
          size="sm"
          className={mainClassName}
          title={waiting ? `${play.label} asked a question` : "Open this run"}
          onClick={() => setTrailOpen(true)}
        >
          {waiting && <CircleHelp className="size-3.5 text-info" aria-hidden />}
          {!waiting && <LoaderCircle className="size-3.5 animate-spin motion-reduce:animate-none text-warning" aria-hidden />}
          {waiting ? "Waiting for your answer" : play.label}
        </Button>
      )}
      {canStop && (
        <Button
          size="icon"
          variant="ghost"
          className="size-7 shrink-0 text-muted-foreground hover:text-foreground"
          title="Stop this run"
          aria-label={`Stop ${play.label}`}
          disabled={stopTrail.isPending}
          onClick={() => stopTrail.mutate(activeTrail.id)}
        >
          <Square className="size-3 fill-current" aria-hidden />
        </Button>
      )}
      {!ownRun && (
        <Button
          variant={variant}
          size="sm"
          className={mainClassName}
          disabled={reason !== "" || blocked === undefined}
          title={reason || play.description || play.label}
          onClick={() => setDialogOpen(true)}
        >
          {outOfDate && <span role="img" aria-label="out of date" className="size-2 rounded-full bg-warning" />}
          {!outOfDate && <Sparkles className="size-3.5" aria-hidden />}
          <span className="min-w-0 truncate">{label ?? play.label}</span>
        </Button>
      )}
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
