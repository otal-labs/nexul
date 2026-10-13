import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { HarnessProject } from "@/models/Pairing";

interface SetupFolderPickProps {
  projects: HarnessProject[];
  folder: string;
  disabled: boolean;
  onPick: (folder: string) => void;
}

// Where the turns run and what they install, under the list; setup only writes user-level files, so any T3 project works.
export const SetupFolderPick = ({ projects, folder, disabled, onPick }: SetupFolderPickProps) => (
  <div className="space-y-1.5 text-xs text-muted-foreground">
    {projects.length > 0 && (
      <div className="flex flex-wrap items-center gap-2">
        <label htmlFor="setup-folder">Runs in</label>
        <Select value={folder} onValueChange={onPick} disabled={disabled}>
          <SelectTrigger id="setup-folder" className="h-7 w-auto max-w-64 px-2 text-xs">
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
        <span className="min-w-0 truncate font-mono">{folder}</span>
      </div>
    )}
    <p>
      Installs mattpocock/skills and nexul-memory into <span className="font-mono">~/.claude/skills/</span> and{" "}
      <span className="font-mono">~/.agents/skills/</span>, never over a skill you already have.
    </p>
  </div>
);
