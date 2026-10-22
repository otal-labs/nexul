import { ComputerFact as Fact } from "@/components/settings/ComputerFact";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { MAX_RUNS_AT_ONCE, MIN_RUNS_AT_ONCE, type Computer } from "@/models/Pairing";

const RUNS = Array.from({ length: MAX_RUNS_AT_ONCE - MIN_RUNS_AT_ONCE + 1 }, (_, i) => String(MIN_RUNS_AT_ONCE + i));
const NO_FOLDER = "__none__";

interface ComputerRunFactsProps {
  computer: Computer;
}

// What a computer takes: how many play runs at once (ADR 0166), and the T3 project @Agent chats outside any project run
// in. Both apply the moment they are picked.
export const ComputerRunFacts = ({ computer }: ComputerRunFactsProps) => {
  const update = useUpdateComputerSettings(computer.id);
  const projects = computer.facts?.projects ?? [];
  const folder = computer.chat_project_id || NO_FOLDER;

  return (
    <>
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
          {RUNS.map((n) => (
            <ToggleGroupItem key={n} value={n} aria-label={n} className="h-6 w-8 px-0 font-mono tabular-nums">
              {n}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Fact>
      <Fact label="Chat folder">
        <Select value={folder} onValueChange={(v) => update.mutate({ chat_project_id: v === NO_FOLDER ? "" : v })}>
          <SelectTrigger aria-label={`Chat folder on ${computer.name}`} className="h-7 w-full max-w-72 text-xs">
            <SelectValue />
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
