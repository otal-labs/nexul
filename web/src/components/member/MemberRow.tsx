import { useState } from "react";
import { UserX } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { Button } from "@/components/ui/button";
import type { Role } from "@/models/Role";

interface MemberRowProps {
  userId: string;
  login: string;
  roleId: string;
  roles: Role[];
  isOwner: boolean;
  isRemoving: boolean;
  onRoleChange: (userId: string, roleId: string) => void;
  onRemove: (userId: string) => void;
  index: number;
}

// Owner reuses the neutral bg-muted role tag, not an editable control — the singleton role never changes here.
const roleChipClass =
  "inline-flex shrink-0 items-center rounded-full bg-muted px-2 py-0.5 font-mono text-[11px] font-medium uppercase tracking-wide text-muted-foreground";

export const MemberRow = ({
  userId,
  login,
  roleId,
  roles,
  isOwner,
  isRemoving,
  onRoleChange,
  onRemove,
  index,
}: MemberRowProps) => {
  const [avatarFailed, setAvatarFailed] = useState(false);
  const enterDelayMs = Math.min(index, 7) * 25;

  return (
    <li className="transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div
        className="animate-in fade-in-0 slide-in-from-bottom-1 flex items-center gap-2 px-3 py-3 duration-150 ease-out sm:gap-3 sm:px-4"
        style={{ animationDelay: `${enterDelayMs}ms` }}
      >
        {avatarFailed && (
          <span
            aria-hidden
            className="flex size-6 shrink-0 items-center justify-center rounded-full bg-accent text-xs font-medium text-accent-foreground"
          >
            {login.charAt(0).toUpperCase()}
          </span>
        )}
        {!avatarFailed && (
          <img
            src={`https://github.com/${login}.png`}
            alt=""
            aria-hidden
            className="size-6 shrink-0 rounded-full bg-accent"
            onError={() => setAvatarFailed(true)}
          />
        )}
        <div className="min-w-0 flex-1">
          <span className="truncate font-medium">{login}</span>
          <p className="truncate font-mono text-xs text-muted-foreground">@{login}</p>
        </div>
        {isOwner && <span className={roleChipClass}>Owner</span>}
        {!isOwner && (
          <Select value={roleId} onValueChange={(value) => onRoleChange(userId, value)}>
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
        )}
        {!isOwner && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Remove ${login}`}
            title="Remove member"
            disabled={isRemoving}
            onClick={() => onRemove(userId)}
          >
            <UserX className="size-4" />
          </Button>
        )}
      </div>
    </li>
  );
};
