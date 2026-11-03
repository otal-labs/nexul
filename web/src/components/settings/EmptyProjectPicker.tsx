import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { HarnessProject } from "@/models/Pairing";

interface EmptyProjectPickerProps {
  computerName: string;
  projects: HarnessProject[];
  onPick: (harnessProjectId: string) => void;
}

// A T3 project with no git remote never matches a repository, so the only way to use one is to pick it here.
export const EmptyProjectPicker = ({ computerName, projects, onPick }: EmptyProjectPickerProps) => {
  const empty = projects.filter((p) => !p.repository);

  return (
    empty.length > 0 && (
      <Select onValueChange={onPick}>
        <SelectTrigger aria-label={`Use an empty project on ${computerName}`} className="h-8 w-64 text-xs">
          <SelectValue placeholder="Use an empty project" />
        </SelectTrigger>
        <SelectContent>
          {empty.map((p) => (
            <SelectItem key={p.id} value={p.id} className="text-xs">
              {p.title} <span className="font-mono text-muted-foreground">{p.path}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )
  );
};
