import { ChevronRight } from "lucide-react";
import { useId } from "react";

import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { ModelChoice } from "@/components/model/ModelChoice";
import { SetupModelText } from "@/components/pairing/SetupModelText";
import { SetupProviderAction } from "@/components/pairing/SetupProviderAction";
import { SetupProviderStatus } from "@/components/pairing/SetupProviderStatus";
import { SetupTranscript } from "@/components/pairing/SetupTranscript";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import type { OptionSetting, SetupProviderRow } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupProviderItemProps {
  row: SetupProviderRow;
  model: string;
  options: OptionSetting[];
  open: boolean;
  busy: boolean;
  stopping: boolean;
  forgetting: boolean;
  onInclude: (provider: string, on: boolean) => void;
  onPick: (provider: string, model: string) => void;
  onOptions: (provider: string, options: OptionSetting[]) => void;
  onRun: (provider: string) => void;
  onStop: (provider: string) => void;
  onForget: (provider: string) => void;
  onToggle: (provider: string) => void;
}

// One provider on one grid: mark, name, the model its turn runs on, its own run and its switch, then its state across the row; the name opens its transcript.
export const SetupProviderItem = ({ row, model, options, open, busy, stopping, forgetting, onInclude, onPick, onOptions, onRun, onStop, onForget, onToggle }: SetupProviderItemProps) => {
  const statusId = useId();
  const running = row.state === "running" || row.state === "queued";
  const listed = !!row.choice;
  // Off or gone, a row has no picker, and its name takes the picker's column; off, the action's too.
  const picking = row.on && listed;
  const nameSpan = !picking && (row.gone ? "col-span-2" : "col-span-3");
  // Steps live only in this tab and arrive a while after a turn starts; until there are some, the state line says it all.
  const kept = useSetupActivityStore((s) => !!row.run?.turnId && (s.steps[row.run.turnId]?.length ?? 0) > 0);
  const transcript = kept ? row.run : undefined;
  const name = (
    <span title={row.name} className={cn("truncate text-sm font-medium", (!row.on || !listed) && "text-muted-foreground")}>
      {row.name}
    </span>
  );
  return (
    <li className="py-3">
      <div className="grid grid-cols-[1rem_minmax(0,1fr)_17rem_5.5rem_2rem] grid-rows-[2rem_auto] items-center gap-x-3 gap-y-1">
        <HarnessProviderMark driver={row.driver} className={cn("size-4 shrink-0", (!row.on || !listed) && "opacity-50")} />
        {transcript && (
          <button
            type="button"
            aria-expanded={open}
            aria-describedby={statusId}
            onClick={() => onToggle(row.provider)}
            className={cn("flex min-w-0 items-center gap-1 justify-self-start rounded-sm text-left", nameSpan)}
          >
            {name}
            <ChevronRight aria-hidden className={cn("size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard", open && "rotate-90")} />
          </button>
        )}
        {!transcript && <span className={cn("flex min-w-0", nameSpan)}>{name}</span>}
        {picking && row.choice && !running && (
          <ModelChoice
            providers={[row.choice.instance]}
            value={{ provider: row.choice.instance.id, model }}
            options={options}
            onPick={(pick) => onPick(row.provider, pick.model)}
            onOptions={(next) => onOptions(row.provider, next)}
            label={`${row.name} model`}
            allowDefault
            mark={false}
            className="w-68"
          />
        )}
        {picking && row.choice && running && <SetupModelText choice={row.choice} model={model} options={options} />}
        {picking && <SetupProviderAction row={row} busy={busy} stopping={stopping} onRun={onRun} onStop={onStop} />}
        {row.gone && (
          <Button type="button" variant="ghost" size="sm" className="justify-self-end text-muted-foreground" loading={forgetting} onClick={() => onForget(row.provider)} aria-label={`Remove ${row.name}`}>
            Remove
          </Button>
        )}
        <Switch
          aria-label={`Use ${row.name}`}
          checked={row.on}
          disabled={running}
          onCheckedChange={(on) => onInclude(row.provider, on)}
          className="justify-self-end disabled:opacity-100"
        />
        <SetupProviderStatus id={statusId} row={row} className="col-span-4 col-start-2" />
      </div>
      {open && transcript && (
        <div className="settle-in pt-2 pl-7">
          <SetupTranscript row={transcript} />
        </div>
      )}
    </li>
  );
};
