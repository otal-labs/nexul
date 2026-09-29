import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { TeamRole } from "@/models/Team";

interface TeamRoleSelectProps {
  label: string;
  roles: TeamRole[];
  value: string;
  disabled?: boolean;
  onChange: (roleId: string) => void;
}

// Only assignable roles are offered: the Owner role is never given or taken here, and the server refuses it anyway.
export const TeamRoleSelect = ({ label, roles, value, disabled = false, onChange }: TeamRoleSelectProps) => (
  <Select value={value} onValueChange={onChange} disabled={disabled}>
    <SelectTrigger aria-label={label} className="h-8 w-auto min-w-28 shrink-0">
      <SelectValue placeholder="Choose a role" />
    </SelectTrigger>
    <SelectContent>
      {roles
        .filter((role) => !role.is_owner)
        .map((role) => (
          <SelectItem key={role.id} value={role.id}>
            {role.name}
          </SelectItem>
        ))}
    </SelectContent>
  </Select>
);
