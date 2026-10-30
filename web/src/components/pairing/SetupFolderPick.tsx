import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { HarnessProject } from "@/models/Pairing";

interface SetupFolderPickProps {
  projects: HarnessProject[];
  folder: string;
  disabled: boolean;
  onPick: (folder: string) => void;
}

const SKILL_DIRS = ["~/.claude/skills/", "~/.agents/skills/"];

// Where the turns run and where they install skills; setup only writes user-level files, so any T3 project works.
export const SetupFolderPick = ({ projects, folder, disabled, onPick }: SetupFolderPickProps) => {
  const pick = projects.length > 0 && (
    <Select value={folder} onValueChange={onPick} disabled={disabled}>
      <SelectTrigger id="setup-folder" className="h-7 w-auto max-w-64 shrink-0 px-2 font-mono text-xs">
        <span className="min-w-0 truncate">
          <SelectValue>{projects.find((p) => p.path === folder)?.title}</SelectValue>
        </span>
      </SelectTrigger>
      <SelectContent className="max-w-(--radix-select-content-available-width)">
        {projects.map((p) => (
          <SelectItem key={p.id} value={p.path} textValue={p.title}>
            <span className="flex min-w-0 flex-col">
              <span className="truncate">{p.title}</span>
              <span className="truncate font-mono text-xs text-muted-foreground">{p.path}</span>
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
  const dirs = SKILL_DIRS.map((d) => (
    <span key={d} className="font-mono whitespace-nowrap text-foreground/80">
      {d}
    </span>
  ));
  return (
    <div className="grid grid-cols-[5rem_minmax(0,1fr)] items-baseline gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
      {pick && <label htmlFor="setup-folder">Runs in</label>}
      {pick && (
        <div className="flex min-w-0 items-baseline gap-2">
          {pick}
          {/* Cut from the start, so the folder's own name, the part that tells two apart, stays. */}
          <span title={folder} className="min-w-0 truncate font-mono [direction:rtl]">
            <bdi>{folder}</bdi>
          </span>
        </div>
      )}
      <span>Installs</span>
      <span className="min-w-0 text-pretty">
        mattpocock/skills and nexul-memory into {dirs[0]} and {dirs[1]}, keeping any skill you already have
      </span>
    </div>
  );
};
