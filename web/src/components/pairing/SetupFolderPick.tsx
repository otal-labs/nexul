import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { HarnessProject } from "@/models/Pairing";

interface SetupFolderPickProps {
  projects: HarnessProject[];
  folder: string;
  disabled: boolean;
  onPick: (folder: string) => void;
}

// The folder the setup turns run in, from the projects T3 Code opens; setup only writes user-level files, so any of them works.
export const SetupFolderPick = ({ projects, folder, disabled, onPick }: SetupFolderPickProps) => (
  <div className="@container space-y-1.5">
    <div className="flex flex-col gap-1.5 @sm:flex-row @sm:items-center @sm:justify-between @sm:gap-3">
      <label htmlFor="setup-folder" className="text-xs font-semibold">
        Folder
      </label>
      <Select value={folder} onValueChange={onPick} disabled={disabled}>
        <SelectTrigger id="setup-folder" className="h-8 @sm:w-60">
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
    </div>
    <p className="text-xs text-muted-foreground">
      Runs in <span className="font-mono break-all">{folder}</span>. Setup only writes user-level files, so any folder T3 Code
      opens works; pick another if this one is gone.
    </p>
  </div>
);
