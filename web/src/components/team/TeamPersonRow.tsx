import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamGapLine } from "@/components/team/TeamGapLine";
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
        className="-mx-3 flex w-[calc(100%+1.5rem)] items-start gap-3 rounded-md px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus"
      >
        {/* Top-aligned like the other multi-line rows: the avatar and chevron sit on the name's 24px line. */}
        <span className="relative -mt-1 shrink-0">
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
          <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            <SettingsStatus tone={person.online ? "success" : "muted"}>{presenceText(person)}</SettingsStatus>
            <AccountStatusLabel status={person.status} />
          </span>
          {/* The 20px chips overhang the 16px line instead of growing it, so both meta lines keep one height. */}
          <span className="flex min-w-0 items-center gap-1.5 overflow-x-clip text-xs whitespace-nowrap text-muted-foreground">
            <TeamSignInAccounts person={person} />
            {only && !several && (
              <span className="-my-0.5 flex min-w-0">
                <RoleChipRow roles={heldRoles(only)} />
              </span>
            )}
            {several && <span className="truncate">{person.workspaces.length} workspaces</span>}
            {!only && <span className="truncate">No workspace</span>}
          </span>
          <TeamGapLine person={person} />
        </span>
        <ChevronRight aria-hidden className="mt-1 size-4 shrink-0 text-muted-foreground" />
      </button>
    </li>
  );
};
