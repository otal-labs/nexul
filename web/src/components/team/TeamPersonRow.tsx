import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamSignInAccounts } from "@/components/team/TeamSignInAccounts";
import { cn } from "@/lib/utils";
import { heldRoles, type TeamPerson } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  onOpen: (id: string) => void;
}

// A name wraps to two lines and keeps its own direction; one workspace shows its roles, several say how many.
export const TeamPersonRow = ({ person, onOpen }: TeamPersonRowProps) => {
  const [only] = person.workspaces;
  const several = person.workspaces.length > 1;
  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(person.id)}
        aria-label={`Open ${personName(person)}`}
        className="flex w-full items-center gap-3 px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:px-4"
      >
        <span className="relative shrink-0">
          <PersonAvatar login={person.login} src={person.avatar_url} className="size-8" />
          <span
            aria-hidden
            data-presence=""
            className={cn(
              "absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full ring-2 ring-card transition-colors duration-150 ease-standard",
              person.online ? "bg-success" : "bg-muted-foreground",
            )}
          />
        </span>
        <span className="min-w-0 flex-1 space-y-1">
          <span dir="auto" className="line-clamp-2 font-medium break-words" title={personName(person)}>
            {personName(person)}
          </span>
          <span className="flex min-w-0 items-center gap-1.5 overflow-hidden text-xs whitespace-nowrap text-muted-foreground">
            <TeamSignInAccounts person={person} />
            {only && !several && <RoleChipRow roles={heldRoles(only)} />}
            {several && <span className="truncate">{person.workspaces.length} workspaces</span>}
            {!only && <span className="truncate">No workspace</span>}
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1 text-right text-sm text-muted-foreground">
          <span className={cn(person.online && "text-foreground")}>{presenceText(person)}</span>
          <AccountStatusLabel status={person.status} />
        </span>
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </button>
    </li>
  );
};
