import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import { TeamSignInAccounts } from "@/components/team/TeamSignInAccounts";
import { cn } from "@/lib/utils";
import type { TeamPerson } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  onOpen: (id: string) => void;
}

// The role in the one workspace, or how many workspaces; the dialog holds the rest.
const membershipText = (person: TeamPerson): string => {
  const [first, ...rest] = person.workspaces;
  if (!first) return "No workspace";
  if (rest.length === 0) return `${first.role_name} in ${first.workspace_name}`;
  return `${person.workspaces.length} workspaces`;
};

// A name wraps to two lines and keeps its own direction, so a long or right-to-left name reads whole.
export const TeamPersonRow = ({ person, onOpen }: TeamPersonRowProps) => (
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
      <span className="min-w-0 flex-1">
        <span dir="auto" className="line-clamp-2 font-medium break-words" title={personName(person)}>
          {personName(person)}
        </span>
        <span className="flex min-w-0 items-center gap-1.5 overflow-hidden text-xs whitespace-nowrap text-muted-foreground">
          <TeamSignInAccounts person={person} />
          <span className="truncate">{membershipText(person)}</span>
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
