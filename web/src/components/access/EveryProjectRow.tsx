import { LayersIcon, LockIcon } from "lucide-react";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { EveryProject } from "@/models/Team";

interface EveryProjectRowProps {
  // Who the row is about, for the control's accessible name.
  subject: string;
  value: EveryProject;
  onChange: (value: EveryProject) => void;
  disabled?: boolean;
}

// The list's lead row, like Every domain on a role: it decides whether the project rows below apply at all.
export const EveryProjectRow = ({ subject, value, onChange, disabled = false }: EveryProjectRowProps) => {
  const restricted = value === EveryProject.None;
  const Icon = restricted ? LockIcon : LayersIcon;
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-2 bg-muted/40 px-2 py-2 @md:px-3">
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 flex-1 basis-48">
        <p className="text-sm font-medium">Every project</p>
        <p className="text-xs text-muted-foreground">
          {restricted ? "Sees only the projects given below. New projects stay hidden." : "Every project at their role's level. Pick Chosen projects to set each one."}
        </p>
      </div>
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label={`Every project for ${subject}`}
        disabled={disabled}
        value={value}
        onValueChange={(next) => next && onChange(next === EveryProject.None ? EveryProject.None : EveryProject.Role)}
      >
        <ToggleGroupItem value={EveryProject.Role}>
          From role
        </ToggleGroupItem>
        <ToggleGroupItem value={EveryProject.None}>
          Chosen projects
        </ToggleGroupItem>
      </ToggleGroup>
    </li>
  );
};
