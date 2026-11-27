import { ZapIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { ComputerPills } from "@/components/agent/ComputerPills";
import { treeRowClass } from "@/components/agent/treeRow";
import { AutoPlaySentence } from "@/components/autoplay/AutoPlaySentence";
import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { useAutoPlayPlaces } from "@/hooks/useAutoPlayPlaces";
import { useOpenRow } from "@/hooks/useOpenRow";
import { useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { cn } from "@/lib/utils";
import { subjectOf, toRequest, type AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

interface AutoPlayTreeRowProps {
  play: Play;
  autoPlay: AutoPlay;
  open: boolean;
}

// An auto play under its play: "⚡ When it …", the computers it would run on, and its own switch; the row unfolds its editor.
export const AutoPlayTreeRow = ({ play, autoPlay, open }: AutoPlayTreeRowProps) => {
  const { toggle } = useOpenRow();
  const update = useUpdateAutoPlay(autoPlay.workspace_id, autoPlay.play_id);
  const places = useAutoPlayPlaces(autoPlay, autoPlay.computer_rule);

  return (
    <div data-off={!autoPlay.enabled || undefined} className={cn(treeRowClass, "py-2 pl-12", open && currentRowFillClass)}>
      <span aria-hidden className="absolute top-0 left-7 h-1/2 w-3.5 rounded-bl-md border-b border-l border-border" />
      {open && <CurrentRowEdge />}
      <button type="button" aria-expanded={open} onClick={() => toggle(autoPlay.id)} className="inset-focus min-w-0 rounded-md py-0.5 text-left sm:col-span-3">
        <span className={cn("flex flex-wrap items-center gap-x-3 gap-y-1.5", !autoPlay.enabled && "text-muted-foreground")}>
          <span className="min-w-0 text-sm [overflow-wrap:anywhere]">
            <ZapIcon aria-hidden className="mr-1.5 mb-0.5 inline size-3.5" />
            <AutoPlaySentence autoPlay={autoPlay} subject={subjectOf(play)} />
          </span>{" "}
          {places && (
            <span className="flex flex-wrap items-center gap-1">
              <ComputerPills places={places} />
            </span>
          )}
        </span>
      </button>
      <ActionSwitch
        permission="autoplays:write"
        verb="switch auto plays on or off"
        checked={autoPlay.enabled}
        onCheckedChange={(enabled) => update.mutate({ id: autoPlay.id, input: { ...toRequest(autoPlay), enabled } })}
        aria-label={`Switch auto play ${autoPlay.id} on or off`}
      />
    </div>
  );
};
