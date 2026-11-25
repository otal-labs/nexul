import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuLabel, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { shownRoles, useComparedRolesStore } from "@/stores/comparedRolesStore";
import type { Role } from "@/models/Role";

const RoleShownItem = ({ role, checked, last }: { role: Role; checked: boolean; last: boolean }) => {
  const toggle = useComparedRolesStore((s) => s.toggle);
  return (
    <DropdownMenuCheckboxItem
      checked={checked}
      disabled={last}
      onCheckedChange={() => toggle(role.id)}
      // Stays open, so several roles change in one visit.
      onSelect={(event) => event.preventDefault()}
      className="data-[disabled]:opacity-100"
    >
      <span className="min-w-0 flex-1 break-words">{role.name}</span>
      {last && <span className="shrink-0 text-xs text-muted-foreground">Last one shown</span>}
    </DropdownMenuCheckboxItem>
  );
};

interface RolesShownMenuProps {
  // Every role Compare roles can show, in the editor's order.
  roles: Role[];
}

// Which roles Compare roles shows, one tick each, kept in this browser; the trigger says how many, so a narrowed matrix never reads as every role.
export const RolesShownMenu = ({ roles }: RolesShownMenuProps) => {
  const hidden = useComparedRolesStore((s) => s.hidden);
  const shown = shownRoles(roles, hidden).map((role) => role.id);
  const label = shown.length === roles.length ? "All roles" : `${shown.length} of ${roles.length} roles`;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="-mr-1.5 h-6 gap-0.5 px-1.5 text-xs font-normal text-muted-foreground" aria-label={`Roles shown: ${label}`}>
          {label}
          <ChevronDownIcon className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-60">
        <DropdownMenuLabel>Roles shown</DropdownMenuLabel>
        {roles.map((role) => (
          <RoleShownItem key={role.id} role={role} checked={shown.includes(role.id)} last={shown.length === 1 && shown.includes(role.id)} />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
