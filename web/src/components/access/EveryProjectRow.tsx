import { LayersIcon, LockIcon } from "lucide-react";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePointerCloseFocus } from "@/hooks/usePointerCloseFocus";
import { EveryProject } from "@/models/Team";

interface EveryProjectRowProps {
  // Who the row is about, for the select's accessible name.
  subject: string;
  value: EveryProject;
  onChange: (value: EveryProject) => void;
  // Read-only: the value reads as text, the way a role does for someone who can't manage members.
  disabled?: boolean;
}

// The control that decides whether the project list exists at all, so it is the full-size one on the row.
export const EveryProjectRow = ({ subject, value, onChange, disabled = false }: EveryProjectRowProps) => {
  const restricted = value === EveryProject.None;
  const Icon = restricted ? LockIcon : LayersIcon;
  const { triggerProps, contentProps } = usePointerCloseFocus<HTMLButtonElement>();
  return (
    <div className="flex min-h-14 items-center gap-3 py-2">
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">Every project</p>
        <p className="text-sm text-muted-foreground">
          {restricted ? "Sees only the projects below. New projects stay hidden." : "Every project, at their role's level."}
        </p>
      </div>
      {disabled && <span className="shrink-0 text-sm text-muted-foreground">{restricted ? "Only chosen projects" : "From role"}</span>}
      {!disabled && (
        <Select value={value} onValueChange={(next) => onChange(next === EveryProject.None ? EveryProject.None : EveryProject.Role)}>
          <SelectTrigger {...triggerProps} aria-label={`Every project for ${subject}`} className="h-9 w-48 shrink-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent align="end" {...contentProps}>
            <SelectItem value={EveryProject.Role}>From role</SelectItem>
            <SelectItem value={EveryProject.None}>Only chosen projects</SelectItem>
          </SelectContent>
        </Select>
      )}
    </div>
  );
};
