import { CrownIcon } from "lucide-react";

import { LevelGlyph } from "@/components/access/LevelGlyph";
import { RoleHolders } from "@/components/settings/RoleHolders";
import type { Role } from "@/models/Role";
import type { TeamPerson } from "@/models/Team";

interface OwnerRoleRowProps {
  role: Role;
  holders: TeamPerson[] | undefined;
}

// The Owner role is protected server-side; its row offers no edit, clone, or delete.
export const OwnerRoleRow = ({ role, holders }: OwnerRoleRowProps) => (
  <li className="space-y-2.5 bg-card px-4 py-3.5">
    <div className="flex items-center gap-3">
      <span className="flex min-w-0 flex-1 items-center gap-1.5">
        <CrownIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <span className="truncate text-sm font-medium" title={role.name}>
          {role.name}
        </span>
      </span>
      {holders && <RoleHolders people={holders} />}
      <span className="w-8" aria-hidden />
    </div>
    <p className="flex items-center gap-1.5 pl-6 text-xs text-muted-foreground">
      <LevelGlyph level={3} />
      <span>
        <span className="text-foreground/90">Every area</span>, always. Can't be renamed, edited, or deleted.
      </span>
    </p>
  </li>
);
