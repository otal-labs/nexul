import { ComputerFact as Fact } from "@/components/settings/ComputerFact";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { ComputerLabels } from "@/components/settings/ComputerLabels";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";
import { RUNS_AT_ONCE_CHOICES, type Computer } from "@/models/Pairing";

const NO_FOLDER = "__none__";

interface ComputerRunFactsProps {
  computer: Computer;
}

// What a computer takes: plays at all (ADR 0211), how many at once (ADR 0166), and where chats outside a project run.
export const ComputerRunFacts = ({ computer }: ComputerRunFactsProps) => {
  const update = useUpdateComputerSettings(computer.id);
  const projects = computer.facts?.projects ?? [];
  const folder = computer.chat_project_id || NO_FOLDER;
  const picked = projects.find((p) => p.id === computer.chat_project_id);

  return (
    <>
      <Fact label="Takes plays">
        <label className="flex items-center gap-2 text-xs">
          <Switch size="sm" checked={!computer.plays_off} disabled={update.isPending} onCheckedChange={(on) => update.mutate({ plays_off: !on })} />
          <span className="text-foreground">Plays, auto plays and @Agent run here</span>
        </label>
        {computer.plays_off && <p className="mt-1.5 text-muted-foreground">Commands, logs and setup still work.</p>}
      </Fact>
      <ComputerLabels computer={computer} />
      <Fact label="Runs at once">
        <ToggleGroup
          type="single"
          variant="segmented"
          size="xs"
          value={String(computer.runs_at_once)}
          // Radix reports "" when the chosen item is clicked again; keep the current count.
          onValueChange={(v) => v && update.mutate({ runs_at_once: Number(v) })}
          aria-label={`Runs at once on ${computer.name}`}
        >
          {RUNS_AT_ONCE_CHOICES.map((n) => (
            <ToggleGroupItem key={n} value={n} aria-label={n} className="h-6 w-8 px-0 font-mono tabular-nums">
              {n}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Fact>
      <Fact label="Chat folder">
        <Select value={folder} onValueChange={(v) => update.mutate({ chat_project_id: v === NO_FOLDER ? "" : v })}>
          <SelectTrigger aria-label={`Chat folder on ${computer.name}`} title={picked?.path} className="h-7 w-full max-w-72 text-xs">
            {/* The list shows each folder's path; the button shows the name alone, so a long path never overflows it. */}
            <SelectValue>
              <span className={cn("min-w-0 truncate", !picked && "text-muted-foreground")}>{picked?.title ?? (computer.chat_project_id ? "Not found" : "None")}</span>
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_FOLDER} className="text-xs text-muted-foreground">
              None
            </SelectItem>
            {projects.map((p) => (
              <SelectItem key={p.id} value={p.id} className="text-xs">
                {p.title} <span className="font-mono text-muted-foreground">{p.path}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="mt-1.5 text-muted-foreground">Where chats outside a project run.</p>
      </Fact>
    </>
  );
};
