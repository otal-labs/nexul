import { ComputerFact as Fact } from "@/components/settings/ComputerFact";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
        <Select value={String(computer.runs_at_once)} onValueChange={(v) => update.mutate({ runs_at_once: Number(v) })}>
          <SelectTrigger aria-label={`Runs at once on ${computer.name}`} className="h-7 w-20 font-mono text-xs tabular-nums">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {RUNS.map((n) => (
              <SelectItem key={n} value={n} className="font-mono text-xs tabular-nums">
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Fact>
      <Fact label="Chat folder">
        <Select value={folder} onValueChange={(v) => update.mutate({ chat_project_id: v === NO_FOLDER ? "" : v })}>
          <SelectTrigger aria-label={`Chat folder on ${computer.name}`} className="h-7 w-full max-w-72 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_FOLDER} className="text-xs text-muted-foreground">
              None: chats outside a project go to your next computer
            </SelectItem>
            {projects.map((p) => (
              <SelectItem key={p.id} value={p.id} className="text-xs">
                {p.title} <span className="font-mono text-muted-foreground">{p.path}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Fact>
    </>
  );
};
