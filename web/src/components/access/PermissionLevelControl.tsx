import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { levelName } from "@/models/PermissionLevel";

interface PermissionLevelControlProps {
  label: string;
  levelCount: number;
  level: number | undefined;
  onLevel: (level: number) => void;
}

// Every rung up to the chosen one fills, so the control reads as "this much access", not "one of these".
export const PermissionLevelControl = ({ label, levelCount, level, onLevel }: PermissionLevelControlProps) => (
  <ToggleGroup
    type="single"
    variant="outline"
    size="sm"
    role="radiogroup"
    aria-label={label}
    value={level === undefined ? "" : String(level)}
    onValueChange={(next) => next && onLevel(Number(next))}
    className="col-span-2 w-full @md:col-span-1 @md:w-fit"
  >
    {Array.from({ length: levelCount + 1 }, (_, rung) => (
      <ToggleGroupItem
        key={rung}
        value={String(rung)}
        className={cn(
          "h-7 flex-1 px-0 text-xs @md:w-14 @md:flex-none text-muted-foreground transition-colors duration-150 ease-standard data-[state=on]:bg-accent data-[state=on]:text-foreground",
          level !== undefined && rung > 0 && rung <= level && "bg-accent text-foreground",
        )}
      >
        {levelName(rung)}
      </ToggleGroupItem>
    ))}
  </ToggleGroup>
);
