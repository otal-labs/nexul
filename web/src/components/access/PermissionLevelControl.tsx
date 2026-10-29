import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { levelName } from "@/models/PermissionLevel";

interface PermissionLevelControlProps {
  label: string;
  levelCount: number;
  level: number | undefined;
  onLevel: (level: number) => void;
  disabled?: boolean;
  // What the first rung is called: plays are seen, not read.
  readName?: string;
}


// Every rung below the chosen one takes the ember wash and the chosen one its hairline, in four fixed 56px slots, None first, so each level holds one x on every row; a shorter ladder leaves its last slots empty.
export const PermissionLevelControl = ({ label, levelCount, level, onLevel, disabled = false, readName = "Read" }: PermissionLevelControlProps) => (
  <span className="col-span-2 flex w-full shrink-0 @md:col-span-1 @md:w-56">
    <ToggleGroup
      type="single"
      variant="outline"
      size="sm"
      role="radiogroup"
      aria-label={label}
      disabled={disabled}
      value={level === undefined ? "" : String(level)}
      onValueChange={(next) => next && onLevel(Number(next))}
      className="w-full @md:w-fit"
    >
      {Array.from({ length: levelCount + 1 }, (_, rung) => (
        <ToggleGroupItem
          key={rung}
          value={String(rung)}
          className={cn(
            "h-7 flex-1 px-0 text-xs text-muted-foreground transition-colors duration-150 ease-standard @md:w-14 @md:flex-none",
            level !== undefined && rung > 0 && rung < level && "bg-brand/10 text-foreground",
            "data-[state=on]:bg-brand/10 data-[state=on]:text-brand data-[state=on]:shadow-[inset_0_0_0_1px_var(--brand)]",
          )}
        >
          {rung === 1 ? readName : levelName(rung)}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  </span>
);
