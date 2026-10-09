import { CrownIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import type { Role } from "@/models/Role";

interface OwnerRoleRowProps {
  role: Role;
}

// The Owner role is protected server-side; its row offers no edit, clone, or delete.
export const OwnerRoleRow = ({ role }: OwnerRoleRowProps) => (
  <li className="flex items-center gap-2 bg-card px-3 py-3">
    <span className="min-w-0 flex-1 truncate text-sm font-medium" title={role.name}>
      {role.name}
    </span>
    <NoFillBadge icon={CrownIcon} color="text-muted-foreground" className="shrink-0">
      Protected role
    </NoFillBadge>
  </li>
);
