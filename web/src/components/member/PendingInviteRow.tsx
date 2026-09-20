import { XIcon } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { Role } from "@/models/Role";

interface PendingInviteRowProps {
  login: string;
  roleId: string;
  roles: Role[];
  onRoleChange: (login: string, roleId: string) => void;
  onRemove: (login: string) => void;
}

// Not yet submitted, so plain local state: a login, a role from this workspace's own catalog, a remove control.
export const PendingInviteRow = ({ login, roleId, roles, onRoleChange, onRemove }: PendingInviteRowProps) => (
  <li className="flex items-center gap-2 rounded-md border bg-card px-3 py-2 sm:gap-3">
    <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-medium text-accent-foreground">
      {login.charAt(0).toUpperCase()}
    </span>
    <span className="min-w-0 flex-1 truncate text-sm font-medium">{login}</span>
    <Select value={roleId} onValueChange={(value) => onRoleChange(login, value)}>
      <SelectTrigger aria-label={`Role for ${login}`} className="h-8 w-auto shrink-0">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {roles.map((role) => (
          <SelectItem key={role.id} value={role.id}>
            {role.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
    <button
      type="button"
      aria-label={`Remove ${login} from invite list`}
      onClick={() => onRemove(login)}
      className="flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground"
    >
      <XIcon className="size-4" />
    </button>
  </li>
);
