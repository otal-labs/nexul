import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { RoleChipRow } from "@/components/role/RoleChipRow";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamGapLine } from "@/components/team/TeamGapLine";
import { TeamSignInAccounts } from "@/components/team/TeamSignInAccounts";
import { heldRoles, type TeamPerson } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  onOpen: (id: string) => void;
}

// Two lines: the name with its presence dot, then one meta line of handles, roles or workspaces, last seen and setup.
export const TeamPersonRow = ({ person, onOpen }: TeamPersonRowProps) => {
  const [only] = person.workspaces;
  const several = person.workspaces.length > 1;
  return (
    <li>
      {/* The button lies under the row's content rather than around it, so the roles' count can be a button of its own. */}
      <div className="relative -mx-2 flex w-[calc(100%+1rem)] items-center gap-3 px-2 py-2.5 text-left [&>:not(button)]:pointer-events-none [&>:not(button)]:relative">
        <button
          type="button"
          onClick={() => onOpen(person.id)}
          aria-label={`Open ${personName(person)}`}
          className="absolute inset-0 rounded-md transition-colors duration-150 ease-standard hover:bg-accent inset-focus"
        />
        <PersonAvatar login={person.login} name={personName(person)} src={person.avatar_url} className="size-8 shrink-0" />
        <span className="min-w-0 flex-1 space-y-0.5">
          <span className="flex min-w-0 items-center gap-2">
            <span dir="auto" className="truncate text-sm font-medium" title={personName(person)}>
              {personName(person)}
            </span>
            {person.online && (
              <span className="flex shrink-0 items-center">
                <span aria-hidden className="size-1.5 rounded-full bg-success" />
                <span className="sr-only">Online</span>
              </span>
            )}
            <AccountStatusLabel status={person.status} />
          </span>
          {/* The 20px chips overhang the 16px line instead of growing it, so the meta line keeps one height. */}
          <span className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-0.5 text-xs whitespace-nowrap text-muted-foreground">
            <TeamSignInAccounts person={person} />
            {only && !several && (
              <span className="-my-0.5 flex min-w-0">
                <RoleChipRow roles={heldRoles(only)} />
              </span>
            )}
            {several && <span className="tabular-nums">{person.workspaces.length} workspaces</span>}
            {!only && <span>No workspace</span>}
            {!person.online && <span className="tabular-nums">{presenceText(person)}</span>}
            <TeamGapLine person={person} />
          </span>
        </span>
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </div>
    </li>
  );
};
