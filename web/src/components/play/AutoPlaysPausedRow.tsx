import { Pause } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { signalActionClass, signalRowClass } from "@/components/play/signalRowClass";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useResumeAutoPlays } from "@/hooks/PlayQueueHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { PlayType } from "@/models/Play";
import { cn } from "@/lib/utils";

interface AutoPlaysPausedRowProps {
  targetType: PlayType;
  targetId: string;
  autoRuns: number;
  // The ticket's developer's login, who may resume beside autoplays:write holders; "" for a doc.
  developer: string;
  onLeave: () => void;
}

export const AutoPlaysPausedRow = ({ targetType, targetId, autoRuns, developer, onLeave }: AutoPlaysPausedRowProps) => {
  const { data: me } = useFetchMe();
  const canWrite = useHasPermission("autoplays:write");
  const resume = useResumeAutoPlays();
  const [leaving, setLeaving] = useState(false);
  const canResume = canWrite || (developer !== "" && me?.user.login === developer);

  const onResume = () => {
    setLeaving(true);
    onLeave();
    resume.mutate({ targetType, targetId }, { onError: () => setLeaving(false) });
  };

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, signalRowClass)}>
      <Pause className="mt-0.5 size-3.5 shrink-0 text-warning" aria-hidden />
      <div className="min-w-0 flex-1">
        <p>Auto plays paused</p>
        <p className="text-muted-foreground">
          <span className="font-mono tabular-nums">{autoRuns}</span> {autoRuns === 1 ? "run" : "runs"} today
        </p>
      </div>
      {canResume && (
        <Button variant="ghost" size="sm" className={signalActionClass} disabled={leaving} onClick={onResume}>
          Resume
        </Button>
      )}
    </li>
  );
};
