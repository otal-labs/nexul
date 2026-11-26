import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { OverflowCount } from "@/components/OverflowCount";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { TeamGapLine } from "@/components/team/TeamGapLine";
import { TeamSignInAccounts } from "@/components/team/TeamSignInAccounts";
import { heldRoles, type TeamPerson, type TeamRole } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  onOpen: (id: string) => void;
}

// The role as plain ink text, like the workspace count beside it on other rows, the rest behind a count.
const TeamRoleNames = ({ roles }: { roles: TeamRole[] }) => {
  const [first, ...rest] = roles;
  return (
    <span className="flex min-w-0 items-baseline gap-1">
      {!first && <span>No roles</span>}
      {first && <span className="truncate text-foreground">{first.name}</span>}
      {rest.length > 0 && (
        <span className="pointer-events-auto">
          <OverflowCount label={`${rest.length} more ${rest.length === 1 ? "role" : "roles"}`} items={rest.map((role) => role.name)} />
        </span>
      )}
    </span>
  );
};

// The name over the handles and one muted setup mark, then two fixed columns every row shares, role or workspaces and
// presence, so the eye scans down them.
export const TeamPersonRow = ({ person, onOpen }: TeamPersonRowProps) => {
  const [only] = person.workspaces;
  const several = person.workspaces.length > 1;
  return (
    <li>
      {/* The button lies under the row's content rather than around it, so the roles' count can be a button of its own. */}
      <div className="relative -mx-2 flex w-[calc(100%+1rem)] items-center gap-4 px-2 py-2.5 text-left [&>:not(button)]:pointer-events-none [&>:not(button)]:relative">
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
            <AccountStatusLabel status={person.status} />
          </span>
          <span className="flex min-w-0 items-center gap-x-4 text-xs whitespace-nowrap text-muted-foreground">
            <TeamSignInAccounts person={person} />
            <TeamGapLine person={person} />
          </span>
        </span>
        <span className="flex w-36 shrink-0 text-xs text-muted-foreground">
          {only && !several && <TeamRoleNames roles={heldRoles(only)} />}
          {several && <span className="tabular-nums">{person.workspaces.length} workspaces</span>}
          {!only && <span>No workspace</span>}
        </span>
        <span className="w-20 shrink-0 text-right text-xs text-muted-foreground tabular-nums">
          {person.online && <SettingsStatus tone="success">Online</SettingsStatus>}
          {!person.online && person.last_seen_at && <span className="sr-only">Last seen </span>}
          {!person.online && (person.last_seen_at ? formatRelativeTime(person.last_seen_at) : presenceText(person))}
        </span>
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </div>
    </li>
  );
};
