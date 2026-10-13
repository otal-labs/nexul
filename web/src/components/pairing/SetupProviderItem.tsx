import { ChevronRight } from "lucide-react";

import { Switch } from "@/components/ui/switch";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { ModelChoice } from "@/components/model/ModelChoice";
import { SetupProviderAction } from "@/components/pairing/SetupProviderAction";
import { SetupStateGlyph } from "@/components/pairing/SetupStateGlyph";
import { SetupTranscript } from "@/components/pairing/SetupTranscript";
import { setupProviderLine, type OptionSetting, type SetupProviderRow } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupProviderItemProps {
  row: SetupProviderRow;
  model: string;
  options: OptionSetting[];
  open: boolean;
  // The first row waiting on the person, lifted so the eye lands on it.
  next: boolean;
  busy: boolean;
  onInclude: (provider: string, on: boolean) => void;
  onPick: (provider: string, model: string) => void;
  onOptions: (provider: string, options: OptionSetting[]) => void;
  onRun: (provider: string) => void;
  onToggle: (provider: string) => void;
}

// One provider: its switch, name over its state, the model its turn runs on, and its own run; the name opens its transcript.
export const SetupProviderItem = ({ row, model, options, open, next, busy, onInclude, onPick, onOptions, onRun, onToggle }: SetupProviderItemProps) => {
  const running = row.state === "running" || row.state === "queued";
  return (
    <li className={cn("transition-colors duration-150 ease-standard", next && "bg-accent/40")}>
      <div className="flex min-h-14 items-center gap-3 px-3 py-2">
        <Switch aria-label={`Use ${row.name}`} checked={row.on} disabled={!row.choice || running} onCheckedChange={(on) => onInclude(row.provider, on)} />
        <HarnessProviderMark driver={row.driver} className={cn("size-4 shrink-0", !row.on && "opacity-50")} />
        <button
          type="button"
          aria-expanded={row.run ? open : undefined}
          disabled={!row.run}
          onClick={() => onToggle(row.provider)}
          className="quiet-focus min-w-0 flex-1 rounded-md text-left focus-visible:outline-2 focus-visible:outline-focus disabled:cursor-default"
        >
          <span className={cn("flex min-w-0 items-center gap-1 text-sm font-medium", !row.on && "text-muted-foreground")}>
            <span className="truncate">{row.name}</span>
            {row.run && (
              <ChevronRight aria-hidden className={cn("size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard", open && "rotate-90")} />
            )}
          </span>
          <span className="mt-0.5 flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
            {row.on && <SetupStateGlyph state={row.state} className="size-3.5" />}
            <span className="truncate" role={running ? "status" : undefined}>
              {setupProviderLine(row)}
            </span>
          </span>
        </button>
        {row.on && row.choice && (
          <ModelChoice
            providers={[row.choice.instance]}
            value={{ provider: row.choice.instance.id, model }}
            options={options}
            onPick={(pick) => onPick(row.provider, pick.model)}
            onOptions={(next) => onOptions(row.provider, next)}
            label={`${row.name} model`}
            allowDefault
            disabled={running}
            className="w-48 shrink-0 @xl:w-60"
          />
        )}
        {row.on && row.choice && <SetupProviderAction row={row} busy={busy} onRun={onRun} />}
      </div>
      {open && row.run && (
        <div className="settle-in px-3 pb-3 pl-[4.75rem]">
          <SetupTranscript row={row.run} />
        </div>
      )}
    </li>
  );
};
