import { useId } from "react";
import { ZapIcon } from "lucide-react";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { ComputerPills } from "@/components/agent/ComputerPills";
import { RowChevron } from "@/components/agent/RowChevron";
import { stopRowClick, treeRowClass } from "@/components/agent/treeRow";
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

// An auto play under its play: "⚡ When it …", the computers it would run on, and its own switch; the whole row opens its editor.
export const AutoPlayTreeRow = ({ play, autoPlay, open }: AutoPlayTreeRowProps) => {
  const { toggle } = useOpenRow();
  const update = useUpdateAutoPlay(autoPlay.workspace_id, autoPlay.play_id);
  const places = useAutoPlayPlaces(autoPlay, autoPlay.computer_rule);
  const id = useId();

  return (
    <div data-off={!autoPlay.enabled || undefined} onClick={() => toggle(autoPlay.id)} className={cn(treeRowClass, "min-h-10 py-1.5 pl-12", open && currentRowFillClass)}>
      <span aria-hidden className="absolute top-0 left-7 h-1/2 w-3.5 rounded-bl-md border-b border-l border-border" />
      {open && <CurrentRowEdge />}
      <button type="button" aria-expanded={open} className="inset-focus min-w-0 rounded-md py-0.5 text-left sm:col-span-2 lg:col-span-3">
        <span className="flex items-start gap-1.5">
          <RowChevron open={open} className="mt-0.5" />
          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
            <span id={id} className={cn("min-w-0 text-row [overflow-wrap:anywhere]", autoPlay.enabled ? "text-foreground" : "text-muted-foreground")}>
              <ZapIcon aria-hidden className="mr-1.5 mb-0.5 inline size-3.5" />
              <AutoPlaySentence autoPlay={autoPlay} subject={subjectOf(play)} />
            </span>{" "}
            {places && (
              <span className="flex flex-wrap items-center gap-1">
                <ComputerPills places={places} />
              </span>
            )}
            {autoPlay.one_at_a_time && <span className="text-xs text-muted-foreground">1 at a time</span>}
          </span>
        </span>
      </button>
      <span onClick={stopRowClick} className="flex justify-end">
        <span id={`${id}-turn`} hidden>
          Turn
        </span>
        <span id={`${id}-off`} hidden>
          on or off
        </span>
        <ActionSwitch
          permission="autoplays:write"
          verb="switch auto plays on or off"
          checked={autoPlay.enabled}
          onCheckedChange={(enabled) => update.mutate({ id: autoPlay.id, input: { ...toRequest(autoPlay), enabled } })}
          aria-label="Turn this auto play on or off"
          labelledBy={`${id}-turn ${id} ${id}-off`}
        />
      </span>
    </div>
  );
};
